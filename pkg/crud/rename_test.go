package crud

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ViBiOh/absto/pkg/filesystem"
	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newRenameService(t *testing.T, files ...string) (*Service, string, chan provider.Event) {
	t.Helper()

	root := t.TempDir()

	for _, file := range files {
		fullpath := filepath.Join(root, file)

		require.NoError(t, os.MkdirAll(filepath.Dir(fullpath), 0o755))
		require.NoError(t, os.WriteFile(fullpath, []byte("content"), 0o600))
	}

	storage, err := filesystem.New(root)
	require.NoError(t, err)

	flagSet := flag.NewFlagSet("renderer", flag.ContinueOnError)
	config := renderer.Flags(flagSet, "")
	require.NoError(t, flagSet.Parse(nil))

	rendererService, err := renderer.New(t.Context(), config, fstest.MapFS{
		"static/robots.txt":    {Data: []byte("")},
		"templates/empty.tmpl": {Data: []byte(`{{ define "empty" }}{{ end }}`)},
	}, nil, nil, nil)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)

	shareMock := mocks.NewShareManager(ctrl)
	shareMock.EXPECT().List().Return(nil).AnyTimes()

	metadataMock := mocks.NewMetadataManager(ctrl)
	metadataMock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(provider.Metadata{}, nil).AnyTimes()

	events := make(chan provider.Event, 10)

	return &Service{
		storage:    storage,
		rawStorage: storage,
		renderer:   rendererService,
		share:      shareMock,
		metadata:   metadataMock,
		pushEvent: func(_ context.Context, event provider.Event) {
			events <- event
		},
	}, root, events
}

func renameForm(name, newName string) url.Values {
	return url.Values{"name": {name}, "newFolder": {"/"}, "newName": {newName}}
}

func serveRename(instance *Service, form url.Values) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()

	instance.Rename(recorder, newScopeRequest(form), provider.Request{Path: "/", CanEdit: true})

	return recorder
}

func entryNames(t *testing.T, directory string) []string {
	t.Helper()

	entries, err := os.ReadDir(directory)
	require.NoError(t, err)

	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}

	return names
}

func isCaseInsensitive(t *testing.T) bool {
	t.Helper()

	directory := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(directory, "Probe"), nil, 0o600))

	_, err := os.Stat(filepath.Join(directory, "probe"))

	return err == nil
}

func TestRenameCaseOnly(t *testing.T) {
	t.Parallel()

	t.Run("file", func(t *testing.T) {
		t.Parallel()

		instance, root, events := newRenameService(t, "Photo.JPG")

		recorder := serveRename(instance, renameForm("Photo.JPG", "photo.jpg"))

		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, []string{"photo.jpg"}, entryNames(t, root))

		select {
		case event := <-events:
			assert.Equal(t, provider.RenameEvent, event.Type)
			assert.Equal(t, "Photo.JPG", event.Item.Name())
			assert.Equal(t, "photo.jpg", event.New.Name())
		case <-time.After(5 * time.Second):
			t.Fatal("no rename event")
		}
	})

	t.Run("directory", func(t *testing.T) {
		t.Parallel()

		instance, root, _ := newRenameService(t, "Photos/a.jpg")

		recorder := serveRename(instance, renameForm("Photos/", "photos"))

		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, []string{"photos"}, entryNames(t, root))
		assert.FileExists(t, filepath.Join(root, "photos", "a.jpg"))
	})

	t.Run("does not overwrite another item that differs only by case", func(t *testing.T) {
		t.Parallel()

		if isCaseInsensitive(t) {
			t.Skip("filesystem is case insensitive, two names that differ only by case can't coexist")
		}

		instance, root, events := newRenameService(t, "Photo.JPG", "photo.jpg")

		recorder := serveRename(instance, renameForm("Photo.JPG", "photo.jpg"))

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.ElementsMatch(t, []string{"Photo.JPG", "photo.jpg"}, entryNames(t, root))
		assert.Empty(t, events)
	})
}

func TestRename(t *testing.T) {
	t.Parallel()

	t.Run("to another name", func(t *testing.T) {
		t.Parallel()

		instance, root, _ := newRenameService(t, "first.txt")

		recorder := serveRename(instance, renameForm("first.txt", "second.txt"))

		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, []string{"second.txt"}, entryNames(t, root))
	})

	t.Run("onto an existing item", func(t *testing.T) {
		t.Parallel()

		instance, root, events := newRenameService(t, "first.txt", "second.txt")

		recorder := serveRename(instance, renameForm("first.txt", "second.txt"))

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.ElementsMatch(t, []string{"first.txt", "second.txt"}, entryNames(t, root))
		assert.Empty(t, events)
	})

	t.Run("to the same name only updates the metadata", func(t *testing.T) {
		t.Parallel()

		instance, root, events := newRenameService(t, "first.txt")

		recorder := serveRename(instance, renameForm("first.txt", "first.txt"))

		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, []string{"first.txt"}, entryNames(t, root))
		assert.Empty(t, events)
	})

	t.Run("a missing item", func(t *testing.T) {
		t.Parallel()

		instance, _, events := newRenameService(t)

		recorder := serveRename(instance, renameForm("missing.txt", "other.txt"))

		assert.Equal(t, http.StatusNotFound, recorder.Code)
		assert.Empty(t, events)
	})
}

func TestCheckTarget(t *testing.T) {
	t.Parallel()

	notExist := absto.ErrNotExist(errors.New("missing"))

	cases := map[string]struct {
		oldPath string
		newPath string
		setup   func(storage *mocks.Storage)
		wantErr bool
	}{
		"free name": {
			"/first.txt",
			"/second.txt",
			func(storage *mocks.Storage) {
				storage.EXPECT().Stat(gomock.Any(), "/second.txt").Return(absto.Item{}, notExist)
			},
			false,
		},
		"name already used": {
			"/first.txt",
			"/second.txt",
			func(storage *mocks.Storage) {
				storage.EXPECT().Stat(gomock.Any(), "/second.txt").Return(absto.Item{NameValue: "second.txt", Pathname: "/second.txt"}, nil)
			},
			true,
		},
		"case only change of a lone item": {
			"/Photo.JPG",
			"/photo.jpg",
			func(storage *mocks.Storage) {
				storage.EXPECT().List(gomock.Any(), "/").Return([]absto.Item{{NameValue: "Photo.JPG"}, {NameValue: "other.txt"}}, nil)
			},
			false,
		},
		"case only change with another item having the new name": {
			"/Photo.JPG",
			"/photo.jpg",
			func(storage *mocks.Storage) {
				storage.EXPECT().List(gomock.Any(), "/").Return([]absto.Item{{NameValue: "Photo.JPG"}, {NameValue: "photo.jpg"}}, nil)
			},
			true,
		},
		"case only change of a directory": {
			"/Albums/Summer/",
			"/Albums/summer/",
			func(storage *mocks.Storage) {
				storage.EXPECT().List(gomock.Any(), "/Albums/").Return([]absto.Item{{NameValue: "Summer", IsDirValue: true}}, nil)
			},
			false,
		},
		"case only change of a directory with another one having the new name": {
			"/Albums/Summer/",
			"/Albums/summer/",
			func(storage *mocks.Storage) {
				storage.EXPECT().List(gomock.Any(), "/Albums/").Return([]absto.Item{{NameValue: "Summer", IsDirValue: true}, {NameValue: "summer", IsDirValue: true}}, nil)
			},
			true,
		},
		"listing failure": {
			"/Photo.JPG",
			"/photo.jpg",
			func(storage *mocks.Storage) {
				storage.EXPECT().List(gomock.Any(), "/").Return(nil, errors.New("unavailable"))
			},
			true,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			storage := mocks.NewStorage(gomock.NewController(t))
			tc.setup(storage)

			err := (&Service{storage: storage}).checkTarget(t.Context(), tc.oldPath, tc.newPath)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
