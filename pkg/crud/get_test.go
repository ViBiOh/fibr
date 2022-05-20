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
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockExif := mocks.NewExif(ctrl)

	mockExif.EXPECT().GetExifFor(gomock.Any(), gomock.Any()).Return(exas.Exif{
		Geocode: exas.Geocode{
			Latitude:  1.0,
			Longitude: 1.0,
		},
		Date: time.Date(2022, 0o2, 22, 22, 0o2, 22, 0, time.UTC),
	}, nil).AnyTimes()

	mockeStorage := mocks.NewStorage(ctrl)

	mockeStorage.EXPECT().Info(gomock.Any(), gomock.Any()).Return(absto.Item{}, nil).AnyTimes()

	instance := App{
		exifApp:    mockExif,
		storageApp: mockeStorage,
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	request := provider.Request{}
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

	for i := 0; i < b.N; i++ {
		instance.serveGeoJSON(httptest.NewRecorder(), r, request, items)
	}
}
