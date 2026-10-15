package crud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ViBiOh/absto/pkg/filesystem"
	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func BenchmarkServeGeoJSON(b *testing.B) {
	items := []absto.Item{
		{
			ID:         "1234",
			NameValue:  "first.jpeg",
			Pathname:   "/first.jpeg",
			Extension:  ".jpeg",
			IsDirValue: false,
		},
		{
			ID:         "5678",
			NameValue:  "second.jpeg",
			Pathname:   "/second.jpeg",
			Extension:  ".jpeg",
			IsDirValue: false,
		},
		{
			ID:         "9012",
			NameValue:  "third.jpeg",
			Pathname:   "/third.jpeg",
			Extension:  ".jpeg",
			IsDirValue: false,
		},
	}

	ctrl := gomock.NewController(b)

	mockExif := mocks.NewMetadataManager(ctrl)

	mockExif.EXPECT().GetAllMetadataFor(gomock.Any(), gomock.Any()).Return(map[string]provider.Metadata{
		"9012": {
			Exif: exas.Exif{
				Geocode: exas.Geocode{
					Latitude:  1.0,
					Longitude: 1.0,
				},
				Date: time.Date(2022, 0o2, 22, 22, 0o2, 22, 0, time.UTC),
			},
		},
		"5678": {
			Exif: exas.Exif{
				Geocode: exas.Geocode{
					Latitude:  1.0,
					Longitude: 1.0,
				},
				Date: time.Date(2022, 0o2, 22, 22, 0o2, 22, 0, time.UTC),
			},
		},
		"1234": {
			Exif: exas.Exif{
				Geocode: exas.Geocode{
					Latitude:  1.0,
					Longitude: 1.0,
				},
				Date: time.Date(2022, 0o2, 22, 22, 0o2, 22, 0, time.UTC),
			},
		},
	}, nil).AnyTimes()

	mockExif.EXPECT().ListDir(gomock.Any(), gomock.Any()).Return(items, nil).AnyTimes()

	instance := Service{
		metadata: mockExif,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	request := provider.Request{}
	item := absto.Item{
		ID:         "1234",
		NameValue:  "first.jpeg",
		Pathname:   "/first.jpeg",
		Extension:  ".jpeg",
		IsDirValue: false,
	}

	for i := 0; i < b.N; i++ {
		instance.serveGeoJSON(httptest.NewRecorder(), r, request, item, items)
	}
}

func TestDichotomicFind(t *testing.T) {
	t.Parallel()

	type args struct {
		items []absto.Item
		id    string
	}

	cases := map[string]struct {
		args args
		want absto.Item
	}{
		"empty": {
			args{},
			absto.Item{},
		},
		"one item": {
			args{
				items: []absto.Item{
					{ID: "1234"},
				},
				id: "1234",
			},
			absto.Item{ID: "1234"},
		},
		"not found": {
			args{
				items: []absto.Item{
					{ID: "1234"},
				},
				id: "2345",
			},
			absto.Item{},
		},
		"odd number lower": {
			args{
				items: []absto.Item{
					{ID: "1234"},
					{ID: "2345"},
					{ID: "3456"},
				},
				id: "1234",
			},
			absto.Item{ID: "1234"},
		},
		"odd number upper": {
			args{
				items: []absto.Item{
					{ID: "1234"},
					{ID: "2345"},
					{ID: "3456"},
				},
				id: "3456",
			},
			absto.Item{ID: "3456"},
		},
		"even number lower": {
			args{
				items: []absto.Item{
					{ID: "1234"},
					{ID: "2345"},
					{ID: "3456"},
					{ID: "4567"},
				},
				id: "2345",
			},
			absto.Item{ID: "2345"},
		},
		"even number upper": {
			args{
				items: []absto.Item{
					{ID: "1234"},
					{ID: "2345"},
					{ID: "3456"},
					{ID: "4567"},
				},
				id: "4567",
			},
			absto.Item{ID: "4567"},
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if got := dichotomicFind(testCase.args.items, testCase.args.id); got != testCase.want {
				t.Errorf("dichotomicFind() = %#v, want %#v", got, testCase.want)
			}
		})
	}
}

func TestIsPassiveContent(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		filename string
		want     bool
	}{
		"jpeg":                   {"photo.jpg", true},
		"png uppercase":          {"PHOTO.PNG", true},
		"webp":                   {"thumbnail.webp", true},
		"svg can embed scripts":  {"vector.svg", false},
		"pdf":                    {"document.pdf", true},
		"html":                   {"page.html", false},
		"xml":                    {"data.xml", false},
		"javascript":             {"script.js", false},
		"plain text":             {"notes.txt", false},
		"no extension":           {"README", false},
		"unknown extension":      {"archive.unknownext", false},
		"double extension":       {"photo.jpg.html", false},
		"hidden file":            {".htaccess", false},
		"extension only in name": {"html", false},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, isPassiveContent(tc.filename))
		})
	}
}

func TestServeFileHeaders(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		filename        string
		wantSandbox     bool
		wantDisposition string
	}{
		"image":     {"photo.jpg", false, "inline; filename=photo.jpg"},
		"pdf":       {"document.pdf", false, "inline; filename=document.pdf"},
		"html":      {"page.html", true, "inline; filename=page.html"},
		"svg":       {"vector.svg", true, "inline; filename=vector.svg"},
		"no ext":    {"payload", true, "inline; filename=payload"},
		"quoted":    {"my page;v2.html", true, `inline; filename="my page;v2.html"`},
		"non ascii": {"résumé.html", true, "inline; filename*=utf-8''r%C3%A9sum%C3%A9.html"},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(root, tc.filename), []byte("<script>alert(1)</script>"), 0o600))

			storage, err := filesystem.New(root)
			require.NoError(t, err)

			item, err := storage.Stat(context.Background(), "/"+tc.filename)
			require.NoError(t, err)

			instance := &Service{storage: storage}
			recorder := httptest.NewRecorder()

			require.NoError(t, instance.serveFile(recorder, httptest.NewRequest(http.MethodGet, "/"+url.PathEscape(tc.filename), nil), item))

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, tc.wantDisposition, recorder.Header().Get("Content-Disposition"))

			if tc.wantSandbox {
				assert.Equal(t, "sandbox", recorder.Header().Get("Content-Security-Policy"))
			} else {
				assert.Empty(t, recorder.Header().Get("Content-Security-Policy"))
			}
		})
	}
}
