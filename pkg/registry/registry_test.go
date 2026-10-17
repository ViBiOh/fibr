package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ViBiOh/absto/pkg/filesystem"
	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testFilename = "/.fibr/items.json"
	testChannel  = "channel"
)

type item struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type controlledStorage struct {
	absto.Storage
	failWrites atomic.Bool
	writeDelay time.Duration
}

func (c *controlledStorage) WriteTo(ctx context.Context, name string, reader io.Reader, opts absto.WriteOpts) error {
	if c.writeDelay != 0 {
		time.Sleep(c.writeDelay)
	}

	if c.failWrites.Load() {
		return errors.New("write failed")
	}

	return c.Storage.WriteTo(ctx, name, reader, opts)
}

type publisher struct {
	messages   []item
	mutex      sync.Mutex
	shouldFail atomic.Bool
}

func (p *publisher) publish(_ context.Context, _ string, value any) error {
	if p.shouldFail.Load() {
		return errors.New("publish failed")
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.messages = append(p.messages, value.(item))

	return nil
}

func (p *publisher) all() []item {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return append([]item(nil), p.messages...)
}

func newStorage(t *testing.T) *controlledStorage {
	t.Helper()

	base, err := filesystem.New(t.TempDir())
	require.NoError(t, err)

	return &controlledStorage{Storage: base}
}

func newRegistry(t *testing.T, storage absto.Storage, exclusiveService exclusive.Service) (*Registry[item], *publisher) {
	t.Helper()

	messages := &publisher{}

	redisClient := mocks.NewRedisClient(gomock.NewController(t))
	redisClient.EXPECT().PublishJSON(gomock.Any(), testChannel, gomock.Any()).DoAndReturn(messages.publish).AnyTimes()

	return New[item](storage, exclusiveService, redisClient, testFilename, testChannel), messages
}

func saved(t *testing.T, storage absto.Storage) map[string]item {
	t.Helper()

	items, err := provider.LoadJSON[map[string]item](t.Context(), storage, testFilename)
	require.NoError(t, err)

	return items
}

func put(entries ...item) func(map[string]item) ([]item, error) {
	return func(items map[string]item) ([]item, error) {
		for _, entry := range entries {
			items[entry.ID] = entry
		}

		return entries, nil
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()

	t.Run("missing file is created", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		instance, _ := newRegistry(t, storage, exclusive.New(nil))

		require.NoError(t, instance.Load(t.Context()))

		assert.NotNil(t, instance.Items())
		assert.Empty(t, instance.Items())
		assert.Empty(t, saved(t, storage))
	})

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		require.NoError(t, storage.Mkdir(t.Context(), provider.MetadataDirectoryName, absto.DirectoryPerm))
		require.NoError(t, provider.SaveJSON(t.Context(), storage, testFilename, map[string]item{"a": {ID: "a", Value: "1"}}))

		instance, _ := newRegistry(t, storage, exclusive.New(nil))

		require.NoError(t, instance.Load(t.Context()))

		assert.Equal(t, map[string]item{"a": {ID: "a", Value: "1"}}, instance.Items())
	})

	t.Run("null content", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".fibr"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, testFilename), []byte("null"), 0o600))

		storage, err := filesystem.New(root)
		require.NoError(t, err)

		instance, _ := newRegistry(t, storage, exclusive.New(nil))

		require.NoError(t, instance.Load(t.Context()))
		require.NoError(t, instance.Update(t.Context(), put(item{ID: "a"})))

		assert.Equal(t, map[string]item{"a": {ID: "a"}}, instance.Items())
	})

	t.Run("invalid content", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".fibr"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, testFilename), []byte("{"), 0o600))

		storage, err := filesystem.New(root)
		require.NoError(t, err)

		instance, _ := newRegistry(t, storage, exclusive.New(nil))

		assert.Error(t, instance.Load(t.Context()))
		assert.Nil(t, instance.Items())
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("saves then publishes in order", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		instance, messages := newRegistry(t, storage, exclusive.New(nil))

		first, second := item{ID: "a", Value: "1"}, item{ID: "b", Value: "2"}

		require.NoError(t, instance.Update(t.Context(), put(first, second)))

		want := map[string]item{"a": first, "b": second}

		assert.Equal(t, want, instance.Items())
		assert.Equal(t, want, saved(t, storage))
		assert.Equal(t, []item{first, second}, messages.all())
	})

	t.Run("starts from the file content, not from the previous snapshot", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		instance, _ := newRegistry(t, storage, exclusive.New(nil))
		other, _ := newRegistry(t, storage, exclusive.New(nil))

		require.NoError(t, instance.Update(t.Context(), put(item{ID: "a"})))
		require.NoError(t, other.Update(t.Context(), put(item{ID: "b"})))

		assert.Equal(t, map[string]item{"a": {ID: "a"}, "b": {ID: "b"}}, saved(t, storage))
		assert.Equal(t, map[string]item{"a": {ID: "a"}, "b": {ID: "b"}}, other.Items())
	})

	t.Run("without change nothing is saved or published but the snapshot is refreshed", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		instance, messages := newRegistry(t, storage, exclusive.New(nil))
		other, _ := newRegistry(t, storage, exclusive.New(nil))

		require.NoError(t, other.Update(t.Context(), put(item{ID: "a"})))

		storage.failWrites.Store(true)

		require.NoError(t, instance.Update(t.Context(), func(map[string]item) ([]item, error) {
			return nil, nil
		}))

		assert.Equal(t, map[string]item{"a": {ID: "a"}}, instance.Items())
		assert.Empty(t, messages.all())
	})

	t.Run("action error leaves everything untouched", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		instance, messages := newRegistry(t, storage, exclusive.New(nil))

		require.NoError(t, instance.Update(t.Context(), put(item{ID: "a"})))

		err := instance.Update(t.Context(), func(items map[string]item) ([]item, error) {
			items["b"] = item{ID: "b"}
			delete(items, "a")

			return []item{{ID: "b"}}, errors.New("rejected")
		})

		assert.EqualError(t, err, "rejected")
		assert.Equal(t, map[string]item{"a": {ID: "a"}}, instance.Items())
		assert.Equal(t, map[string]item{"a": {ID: "a"}}, saved(t, storage))
		assert.Equal(t, []item{{ID: "a"}}, messages.all())
	})

	t.Run("failed save is not visible and does not block next updates", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		instance, messages := newRegistry(t, storage, exclusive.New(nil))

		require.NoError(t, instance.Update(t.Context(), put(item{ID: "a"})))

		storage.failWrites.Store(true)

		require.Error(t, instance.Update(t.Context(), put(item{ID: "b"})))

		assert.Equal(t, map[string]item{"a": {ID: "a"}}, instance.Items())
		assert.Equal(t, map[string]item{"a": {ID: "a"}}, saved(t, storage))
		assert.Equal(t, []item{{ID: "a"}}, messages.all())

		storage.failWrites.Store(false)

		require.NoError(t, instance.Update(t.Context(), put(item{ID: "c"})))

		assert.Equal(t, map[string]item{"a": {ID: "a"}, "c": {ID: "c"}}, instance.Items())
	})

	t.Run("failed publish still reports the saved state", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		instance, messages := newRegistry(t, storage, exclusive.New(nil))

		messages.shouldFail.Store(true)

		require.Error(t, instance.Update(t.Context(), put(item{ID: "a"})))

		assert.Equal(t, map[string]item{"a": {ID: "a"}}, instance.Items())
		assert.Equal(t, map[string]item{"a": {ID: "a"}}, saved(t, storage))
	})
}

func TestReadersDoNotWaitForWriters(t *testing.T) {
	t.Parallel()

	storage := newStorage(t)
	storage.writeDelay = 300 * time.Millisecond

	instance, _ := newRegistry(t, storage, exclusive.New(nil))
	require.NoError(t, instance.Load(t.Context()))

	done := make(chan error, 1)

	go func() {
		done <- instance.Update(t.Context(), put(item{ID: "a"}))
	}()

	time.Sleep(100 * time.Millisecond)

	begin := time.Now()
	during := instance.Items()
	waited := time.Since(begin)

	assert.Less(t, waited, 50*time.Millisecond)
	assert.Empty(t, during)

	require.NoError(t, <-done)

	assert.Equal(t, map[string]item{"a": {ID: "a"}}, instance.Items())
	assert.Empty(t, during, "a published snapshot must never change")
}

func TestConcurrentUpdates(t *testing.T) {
	t.Parallel()

	storage := newStorage(t)
	instance, messages := newRegistry(t, storage, exclusive.New(nil))
	require.NoError(t, instance.Load(t.Context()))

	const count = 40

	var wg sync.WaitGroup

	for index := range count {
		wg.Add(2)

		go func() {
			defer wg.Done()

			assert.NoError(t, instance.Update(t.Context(), put(item{ID: fmt.Sprintf("item-%d", index)})))
		}()

		go func() {
			defer wg.Done()

			_ = instance.Items()
		}()
	}

	wg.Wait()

	assert.Len(t, instance.Items(), count)
	assert.Len(t, saved(t, storage), count)
	assert.Len(t, messages.all(), count)
}

func TestUpdatesAcrossInstances(t *testing.T) {
	t.Parallel()

	var lock sync.Mutex

	redisClient := mocks.NewRedisClient(gomock.NewController(t))
	redisClient.EXPECT().Enabled().Return(true).AnyTimes()
	redisClient.EXPECT().Exclusive(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, _ string, _ time.Duration, action func(context.Context) error) (bool, error) {
		lock.Lock()
		defer lock.Unlock()

		return true, action(ctx)
	}).AnyTimes()

	storage := newStorage(t)

	const instances, perInstance = 4, 15

	registries := make([]*Registry[item], instances)
	for index := range registries {
		registries[index], _ = newRegistry(t, storage, exclusive.New(redisClient))
	}

	var wg sync.WaitGroup

	for instanceIndex, instance := range registries {
		for index := range perInstance {
			wg.Add(1)

			go func() {
				defer wg.Done()

				assert.NoError(t, instance.Update(t.Context(), put(item{ID: fmt.Sprintf("instance-%d-item-%d", instanceIndex, index)})))
			}()
		}
	}

	wg.Wait()

	assert.Len(t, saved(t, storage), instances*perInstance, "an update has been lost")
}

func TestPatch(t *testing.T) {
	t.Parallel()

	storage := newStorage(t)
	instance, messages := newRegistry(t, storage, exclusive.New(nil))
	require.NoError(t, instance.Update(t.Context(), put(item{ID: "a", Value: "1"})))

	before := instance.Items()

	instance.Patch(func(items map[string]item) {
		items["a"] = item{ID: "a", Value: "2"}
		items["b"] = item{ID: "b"}
	})

	assert.Equal(t, map[string]item{"a": {ID: "a", Value: "1"}}, before, "a published snapshot must never change")
	assert.Equal(t, map[string]item{"a": {ID: "a", Value: "2"}, "b": {ID: "b"}}, instance.Items())
	assert.Equal(t, map[string]item{"a": {ID: "a", Value: "1"}}, saved(t, storage))
	assert.Len(t, messages.all(), 1)

	empty, _ := newRegistry(t, storage, exclusive.New(nil))
	empty.Patch(func(items map[string]item) { items["z"] = item{ID: "z"} })

	assert.Equal(t, map[string]item{"z": {ID: "z"}}, empty.Items())
}
