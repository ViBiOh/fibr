package provider

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ViBiOh/absto/pkg/filesystem"
	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errWrite = errors.New("write failed")

type writeFailingStorage struct {
	absto.Storage
	readBytes int
}

func (w writeFailingStorage) WriteTo(_ context.Context, _ string, reader io.Reader, _ absto.WriteOpts) error {
	if w.readBytes > 0 {
		if _, err := io.CopyN(io.Discard, reader, int64(w.readBytes)); err != nil {
			return err
		}
	}

	return errWrite
}

func saveWithTimeout(t *testing.T, storage absto.Storage, filename string, content any) error {
	t.Helper()

	done := make(chan error, 1)

	go func() {
		done <- SaveJSON(context.Background(), storage, filename, content)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("SaveJSON did not return")

		return nil
	}
}

func TestSaveJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	storage, err := filesystem.New(root)
	require.NoError(t, err)

	t.Run("round trip", func(t *testing.T) {
		t.Parallel()

		content := map[string]string{"first": "1", "second": "2"}

		require.NoError(t, saveWithTimeout(t, storage, "/content.json", content))

		loaded, err := LoadJSON[map[string]string](t.Context(), storage, "/content.json")
		require.NoError(t, err)

		assert.Equal(t, content, loaded)
	})

	t.Run("storage fails before reading", func(t *testing.T) {
		t.Parallel()

		err := saveWithTimeout(t, writeFailingStorage{Storage: storage}, "/content.json", map[string]string{"first": "1"})

		assert.ErrorIs(t, err, errWrite)
	})

	t.Run("storage fails while reading", func(t *testing.T) {
		t.Parallel()

		err := saveWithTimeout(t, writeFailingStorage{Storage: storage, readBytes: 2}, "/content.json", map[string]string{"first": "1"})

		assert.ErrorIs(t, err, errWrite)
	})

	t.Run("content that can't be encoded", func(t *testing.T) {
		t.Parallel()

		err := saveWithTimeout(t, storage, "/unsupported.json", map[string]any{"channel": make(chan int)})

		assert.Error(t, err)
	})
}

func TestLoadJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	storage, err := filesystem.New(root)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(root, "invalid.json"), []byte("{"), 0o600))

	_, err = LoadJSON[map[string]string](t.Context(), storage, "/invalid.json")
	assert.Error(t, err)

	_, err = LoadJSON[map[string]string](t.Context(), storage, "/missing.json")
	assert.True(t, absto.IsNotExist(err))
}
