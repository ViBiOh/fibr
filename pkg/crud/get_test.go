package crud

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/golang/mock/gomock"
)

func BenchmarkServeGeoJSON(b *testing.B) {
	items := []absto.Item{
		{
			ID:        "1234",
			Name:      "first.jpeg",
			Pathname:  "/first.jpeg",
			Extension: ".jpeg",
			IsDir:     false,
		},
		{
			ID:        "5678",
			Name:      "second.jpeg",
			Pathname:  "/second.jpeg",
			Extension: ".jpeg",
			IsDir:     false,
		},
		{
			ID:        "9012",
			Name:      "third.jpeg",
			Pathname:  "/third.jpeg",
			Extension: ".jpeg",
			IsDir:     false,
		},
	}

	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

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

	instance := App{
		metadataApp: mockExif,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	request := provider.Request{}
	item := absto.Item{
		ID:        "1234",
		Name:      "first.jpeg",
		Pathname:  "/first.jpeg",
		Extension: ".jpeg",
		IsDir:     false,
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
		intention, testCase := intention, testCase

		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if got := dichotomicFind(testCase.args.items, testCase.args.id); got != testCase.want {
				t.Errorf("dichotomicFind() = %#v, want %#v", got, testCase.want)
			}
		})
	}
}
