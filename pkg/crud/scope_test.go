package crud

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var scopeFiles = []string{"photos/pic.jpg", "secret.txt", ".fibr/shares.json"}

func newScopeService(t *testing.T) (*Service, string) {
	t.Helper()

	root := t.TempDir()

	for _, file := range scopeFiles {
		fullpath := filepath.Join(root, file)

		require.NoError(t, os.MkdirAll(filepath.Dir(fullpath), 0o755))
		require.NoError(t, os.WriteFile(fullpath, []byte("content"), 0o600))
	}

	storage, err := filesystem.New(root)
	require.NoError(t, err)

	return &Service{storage: storage, rawStorage: storage}, root
}

func newScopeRequest(form url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/plain")

	return req
}

func assertScopeFilesExist(t *testing.T, root string) {
	t.Helper()

	for _, file := range scopeFiles {
		assert.FileExists(t, filepath.Join(root, file))
	}
}

func TestDeleteOutOfScope(t *testing.T) {
	t.Parallel()

	shareRequest := provider.Request{
		Path:    "/",
		CanEdit: true,
		Share:   provider.Share{ID: "abcdef12", Path: "/photos/", Edit: true},
	}

	cases := map[string]struct {
		request    provider.Request
		form       url.Values
		wantStatus int
	}{
		"no name": {
			provider.Request{Path: "/", CanEdit: true},
			url.Values{},
			http.StatusBadRequest,
		},
		"empty name": {
			provider.Request{Path: "/", CanEdit: true},
			url.Values{"name": {""}},
			http.StatusBadRequest,
		},
		"root": {
			provider.Request{Path: "/", CanEdit: true},
			url.Values{"name": {"/"}},
			http.StatusForbidden,
		},
		"dot at root": {
			provider.Request{Path: "/", CanEdit: true},
			url.Values{"name": {"."}},
			http.StatusForbidden,
		},
		"dot in share": {
			shareRequest,
			url.Values{"name": {"./"}},
			http.StatusForbidden,
		},
		"metadata directory": {
			provider.Request{Path: "/", CanEdit: true},
			url.Values{"name": {".fibr/"}},
			http.StatusForbidden,
		},
		"traversal out of share": {
			shareRequest,
			url.Values{"name": {"../secret.txt"}},
			http.StatusForbidden,
		},
		"metadata directory out of share": {
			shareRequest,
			url.Values{"name": {"../.fibr/"}},
			http.StatusForbidden,
		},
		"read only": {
			provider.Request{Path: "/"},
			url.Values{"name": {"secret.txt"}},
			http.StatusForbidden,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			instance, root := newScopeService(t)
			recorder := httptest.NewRecorder()

			instance.Delete(recorder, newScopeRequest(tc.form), tc.request)

			assert.Equal(t, tc.wantStatus, recorder.Code)
			assertScopeFilesExist(t, root)
		})
	}
}

func TestRenameOutOfScope(t *testing.T) {
	t.Parallel()

	shareRequest := provider.Request{
		Path:    "/",
		CanEdit: true,
		Share:   provider.Share{ID: "abcdef12", Path: "/photos/", Edit: true},
	}

	cases := map[string]struct {
		request    provider.Request
		form       url.Values
		wantStatus int
	}{
		"no name": {
			provider.Request{Path: "/", CanEdit: true},
			url.Values{"newFolder": {"/"}, "newName": {"moved"}},
			http.StatusBadRequest,
		},
		"root": {
			provider.Request{Path: "/", CanEdit: true},
			url.Values{"name": {"/"}, "newFolder": {"/"}, "newName": {"moved"}},
			http.StatusForbidden,
		},
		"dot in share": {
			shareRequest,
			url.Values{"name": {"./"}, "newFolder": {"/"}, "newName": {"moved"}},
			http.StatusForbidden,
		},
		"metadata directory": {
			provider.Request{Path: "/", CanEdit: true},
			url.Values{"name": {".fibr/"}, "newFolder": {"/"}, "newName": {"moved"}},
			http.StatusForbidden,
		},
		"traversal out of share": {
			shareRequest,
			url.Values{"name": {"../secret.txt"}, "newFolder": {"/"}, "newName": {"moved.txt"}},
			http.StatusForbidden,
		},
		"read only": {
			provider.Request{Path: "/"},
			url.Values{"name": {"secret.txt"}, "newFolder": {"/"}, "newName": {"moved.txt"}},
			http.StatusForbidden,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			instance, root := newScopeService(t)
			recorder := httptest.NewRecorder()

			instance.Rename(recorder, newScopeRequest(tc.form), tc.request)

			assert.Equal(t, tc.wantStatus, recorder.Code)
			assertScopeFilesExist(t, root)
			assert.NoFileExists(t, filepath.Join(root, "moved"))
			assert.NoFileExists(t, filepath.Join(root, "photos", "moved.txt"))
		})
	}
}
