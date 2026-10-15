package provider

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ViBiOh/absto/pkg/filesystem"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("broken reader")
}

func TestWriteToStorage(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		target          string
		prepare         func(t *testing.T, root string)
		reader          io.Reader
		wantErr         bool
		wantExistingKey string
	}{
		"failed write on a directory keeps its content": {
			"/holidays",
			func(t *testing.T, root string) {
				t.Helper()

				if err := os.MkdirAll(filepath.Join(root, "holidays"), 0o755); err != nil {
					t.Fatal(err)
				}

				if err := os.WriteFile(filepath.Join(root, "holidays", "photo.jpg"), []byte("content"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			failingReader{},
			true,
			"holidays/photo.jpg",
		},
		"failed write removes the partial file": {
			"/holidays/photo.jpg",
			func(t *testing.T, root string) {
				t.Helper()

				if err := os.MkdirAll(filepath.Join(root, "holidays"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			failingReader{},
			true,
			"",
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			testCase.prepare(t, root)

			storage, err := filesystem.New(root)
			if err != nil {
				t.Fatal(err)
			}

			gotErr := WriteToStorage(context.Background(), storage, testCase.target, -1, testCase.reader)
			if (gotErr != nil) != testCase.wantErr {
				t.Errorf("WriteToStorage() error = %v, wantErr %t", gotErr, testCase.wantErr)
			}

			if len(testCase.wantExistingKey) != 0 {
				if _, err := os.Stat(filepath.Join(root, testCase.wantExistingKey)); err != nil {
					t.Errorf("`%s` should still exist: %s", testCase.wantExistingKey, err)
				}
			}

			if testCase.target == "/holidays/photo.jpg" {
				if _, err := os.Stat(filepath.Join(root, "holidays", "photo.jpg")); !os.IsNotExist(err) {
					t.Errorf("partial file should have been removed, stat error = %v", err)
				}
			}
		})
	}
}
