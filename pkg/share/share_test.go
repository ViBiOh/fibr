package share

import (
	"sync"
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

var createdAt = time.Date(2021, 5, 1, 12, 0, 0, 0, time.UTC)

type publishedMessages struct {
	messages []provider.Share
	mutex    sync.Mutex
}

func (p *publishedMessages) all() []provider.Share {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return append([]provider.Share(nil), p.messages...)
}

func newTestService(t *testing.T, shares ...provider.Share) (*Service, absto.Storage, *publishedMessages) {
	t.Helper()

	storage, err := filesystem.New(t.TempDir())
	require.NoError(t, err)

	published := &publishedMessages{}

	redisClient := mocks.NewRedisClient(gomock.NewController(t))
	redisClient.EXPECT().PublishJSON(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, _ string, value any) error {
		published.mutex.Lock()
		defer published.mutex.Unlock()

		published.messages = append(published.messages, value.(provider.Share))

		return nil
	}).AnyTimes()

	instance, err := New(&Config{PubsubChannel: "channel"}, nil, storage, redisClient, exclusive.New(nil))
	require.NoError(t, err)

	instance.clock = func() time.Time { return time.Date(2021, 5, 1, 14, 0, 0, 0, time.UTC) }

	if len(shares) != 0 {
		seed := make(map[string]provider.Share, len(shares))
		for _, share := range shares {
			seed[share.ID] = share
		}

		require.NoError(t, storage.Mkdir(t.Context(), provider.MetadataDirectoryName))
		require.NoError(t, provider.SaveJSON(t.Context(), storage, shareFilename, seed))
	}

	require.NoError(t, instance.shares.Load(t.Context()))

	return instance, storage, published
}

func loadShares(t *testing.T, storage absto.Storage) map[string]provider.Share {
	t.Helper()

	shares, err := provider.LoadJSON[map[string]provider.Share](t.Context(), storage, shareFilename)
	require.NoError(t, err)

	return shares
}

func TestPurgeExpired(t *testing.T) {
	t.Parallel()

	now := time.Date(2021, 5, 1, 14, 0, 0, 0, time.UTC)
	created := time.Date(2021, 5, 1, 12, 0, 0, 0, time.UTC)

	cases := map[string]struct {
		shares     map[string]provider.Share
		want       map[string]provider.Share
		wantPurged []provider.Share
	}{
		"empty": {
			map[string]provider.Share{},
			map[string]provider.Share{},
			nil,
		},
		"purge at boundaries": {
			map[string]provider.Share{
				"1":  {ID: "1", Created: created, Duration: time.Hour},
				"2":  {ID: "2", Created: created, Duration: time.Hour * 24},
				"22": {ID: "22", Created: created, Duration: 0},
				"3":  {ID: "3", Created: created, Duration: time.Hour},
			},
			map[string]provider.Share{
				"2":  {ID: "2", Created: created, Duration: time.Hour * 24},
				"22": {ID: "22", Created: created, Duration: 0},
			},
			[]provider.Share{{ID: "1"}, {ID: "3"}},
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			purged := purgeExpired(tc.shares, now)

			assert.Equal(t, tc.want, tc.shares)
			assert.ElementsMatch(t, tc.wantPurged, purged)
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	first := provider.Share{ID: "abcd1234", Path: "/first/", Created: createdAt}
	second := provider.Share{ID: "efgh5678", Path: "/second/", Created: createdAt}

	instance, _, _ := newTestService(t, first, second)

	cases := map[string]struct {
		pathname string
		want     provider.Share
	}{
		"id with item": {
			"/abcd1234/photo.jpg",
			first,
		},
		"id alone": {
			"/abcd1234",
			first,
		},
		"id directory": {
			"/efgh5678/",
			second,
		},
		"without leading slash": {
			"abcd1234/photo.jpg",
			first,
		},
		"id as prefix of another segment": {
			"/abcd1234-private/photo.jpg",
			provider.Share{},
		},
		"unknown": {
			"/ffffffff/photo.jpg",
			provider.Share{},
		},
		"root": {
			"/",
			provider.Share{},
		},
		"empty": {
			"",
			provider.Share{},
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, instance.Get(tc.pathname))
		})
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	instance, storage, published := newTestService(t)

	id, err := instance.Create(t.Context(), "/photos/", true, false, "hashed", true, time.Hour)
	require.NoError(t, err)
	assert.Len(t, id, 8)

	want := provider.Share{
		ID:       id,
		Path:     "/photos/",
		RootName: "photos",
		Edit:     true,
		Password: "hashed",
		Created:  time.Date(2021, 5, 1, 14, 0, 0, 0, time.UTC),
		Duration: time.Hour,
	}

	assert.Equal(t, want, instance.Get("/"+id+"/photo.jpg"))
	assert.Equal(t, []provider.Share{want}, instance.List())
	assert.Equal(t, map[string]provider.Share{id: want}, loadShares(t, storage))
	assert.Equal(t, []provider.Share{want}, published.all())
}

func TestDelete(t *testing.T) {
	t.Parallel()

	keep := provider.Share{ID: "keep1234", Path: "/keep/", Created: createdAt}
	remove := provider.Share{ID: "gone5678", Path: "/gone/", Created: createdAt}

	instance, storage, published := newTestService(t, keep, remove)

	require.NoError(t, instance.Delete(t.Context(), remove.ID))

	assert.Equal(t, provider.Share{}, instance.Get("/gone5678/"))
	assert.Equal(t, keep, instance.Get("/keep1234/"))
	assert.Equal(t, map[string]provider.Share{keep.ID: keep}, loadShares(t, storage))
	assert.Equal(t, []provider.Share{{ID: remove.ID}}, published.all())
}

func TestUpdatePassword(t *testing.T) {
	t.Parallel()

	existing := provider.Share{ID: "abcd1234", Path: "/photos/", Password: "old", Created: createdAt}

	instance, storage, published := newTestService(t, existing)

	require.NoError(t, instance.UpdatePassword(t.Context(), existing.ID, "secret"))

	updated := instance.Get("/abcd1234/")
	assert.NotEqual(t, "old", updated.Password)
	assert.Contains(t, updated.Password, "$argon2id")
	assert.Equal(t, updated, loadShares(t, storage)[existing.ID])
	assert.Equal(t, []provider.Share{updated}, published.all())

	assert.Error(t, instance.UpdatePassword(t.Context(), "unknown", "secret"))
	assert.Equal(t, provider.Share{}, instance.Get("/unknown/"))
	assert.Len(t, published.all(), 1)
}

func TestRenameItem(t *testing.T) {
	t.Parallel()

	inside := provider.Share{ID: "inside12", Path: "/old/photos/", Created: createdAt}
	root := provider.Share{ID: "root1234", Path: "/old/", Created: createdAt}
	other := provider.Share{ID: "other123", Path: "/other/", Created: createdAt}

	instance, storage, published := newTestService(t, inside, root, other)

	require.NoError(t, instance.renameItem(t.Context(), absto.Item{Pathname: "/old/"}, absto.Item{Pathname: "/new/"}))

	renamedInside := inside
	renamedInside.Path = "/new/photos/"

	renamedRoot := root
	renamedRoot.Path = "/new/"

	want := map[string]provider.Share{inside.ID: renamedInside, root.ID: renamedRoot, other.ID: other}

	assert.Equal(t, want, loadShares(t, storage))
	assert.Equal(t, renamedInside, instance.Get("/inside12/"))
	assert.Equal(t, other, instance.Get("/other123/"))
	assert.ElementsMatch(t, []provider.Share{renamedInside, renamedRoot}, published.all())
}

func TestDeleteItem(t *testing.T) {
	t.Parallel()

	inside := provider.Share{ID: "inside12", Path: "/old/photos/", Created: createdAt}
	root := provider.Share{ID: "root1234", Path: "/old/", Created: createdAt}
	other := provider.Share{ID: "other123", Path: "/other/", Created: createdAt}

	instance, storage, published := newTestService(t, inside, root, other)

	require.NoError(t, instance.deleteItem(t.Context(), absto.Item{Pathname: "/old/"}))

	assert.Equal(t, map[string]provider.Share{other.ID: other}, loadShares(t, storage))
	assert.Equal(t, provider.Share{}, instance.Get("/inside12/"))
	assert.Equal(t, other, instance.Get("/other123/"))
	assert.ElementsMatch(t, []provider.Share{{ID: inside.ID}, {ID: root.ID}}, published.all())

	require.NoError(t, instance.deleteItem(t.Context(), absto.Item{Pathname: "/nothing/"}))
	assert.Len(t, published.all(), 2)
}

func TestPubSubHandle(t *testing.T) {
	t.Parallel()

	existing := provider.Share{ID: "abcd1234", Path: "/photos/", Created: createdAt}
	added := provider.Share{ID: "efgh5678", Path: "/videos/", Created: createdAt}

	instance, _, _ := newTestService(t, existing)

	instance.PubSubHandle(added, nil)
	assert.Equal(t, added, instance.Get("/efgh5678/"))
	assert.Equal(t, existing, instance.Get("/abcd1234/"))

	instance.PubSubHandle(provider.Share{ID: existing.ID}, nil)
	assert.Equal(t, provider.Share{}, instance.Get("/abcd1234/"))
	assert.Equal(t, added, instance.Get("/efgh5678/"))

	instance.PubSubHandle(provider.Share{}, assert.AnError)
	assert.Equal(t, added, instance.Get("/efgh5678/"))
}

func TestCleanShares(t *testing.T) {
	t.Parallel()

	created := time.Date(2021, 5, 1, 12, 0, 0, 0, time.UTC)

	expired := provider.Share{ID: "expired1", Path: "/a/", Created: created, Duration: time.Hour}
	valid := provider.Share{ID: "valid123", Path: "/b/", Created: created, Duration: time.Hour * 24}

	instance, storage, published := newTestService(t, expired, valid)

	require.NoError(t, instance.cleanShares(t.Context()))

	assert.Equal(t, map[string]provider.Share{valid.ID: valid}, loadShares(t, storage))
	assert.Equal(t, provider.Share{}, instance.Get("/expired1/"))
	assert.Equal(t, []provider.Share{{ID: expired.ID}}, published.all())
}
