package thumbnail

import (
	"context"
	"testing"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/httputils/v4/pkg/cache"
	"go.uber.org/mock/gomock"
)

func TestCanHaveThumbnail(t *testing.T) {
	cases := map[string]struct {
		instance Service
		input    absto.Item
		want     bool
	}{
		"empty": {
			Service{},
			absto.Item{},
			false,
		},
		"image": {
			Service{},
			absto.Item{
				NameValue: "test.png",
				Extension: ".png",
			},
			true,
		},
		"pdf": {
			Service{},
			absto.Item{
				NameValue: "test.pdf",
				Extension: ".pdf",
			},
			true,
		},
		"video": {
			Service{},
			absto.Item{
				NameValue: "test.avi",
				Extension: ".avi",
			},
			true,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			if result := tc.instance.CanHaveThumbnail(tc.input); result != tc.want {
				t.Errorf("CanHaveThumbnail() = %t, want %t", result, tc.want)
			}
		})
	}
}

func TestHasThumbnail(t *testing.T) {
	cases := map[string]struct {
		instance Service
		input    absto.Item
		want     bool
	}{
		"not found": {
			Service{},
			absto.Item{
				Pathname:   "path/to/error",
				IsDirValue: true,
			},
			false,
		},
		"found": {
			Service{},
			absto.Item{
				Pathname: "path/to/valid",
			},
			true,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockStorage := mocks.NewStorage(ctrl)
			mockRedisClient := mocks.NewRedisClient(ctrl)
			mockRedisClient.EXPECT().Enabled().Return(false)

			tc.instance.storage = mockStorage
			tc.instance.cache = cache.New(mockRedisClient, nil, mockStorage.Stat, nil)

			if intention == "found" {
				mockStorage.EXPECT().Stat(gomock.Any(), gomock.Any()).Return(absto.Item{}, nil)
			}

			if result := tc.instance.HasThumbnail(context.TODO(), tc.input, SmallSize); result != tc.want {
				t.Errorf("HasThumbnail() = %t, want %t", result, tc.want)
			}
		})
	}
}

func TestPathForScale(t *testing.T) {
	cases := map[string]struct {
		instance Service
		input    absto.Item
		want     string
	}{
		"simple": {
			Service{},
			absto.Item{
				ID:       "dd29ecf524b030a65261e3059c48ab9e1ecb2585",
				Pathname: "/path/to/file.png",
			},
			"/.fibr/path/to/dd29ecf524b030a65261e3059c48ab9e1ecb2585.webp",
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			if result := tc.instance.PathForScale(tc.input, SmallSize); result != tc.want {
				t.Errorf("PathForScale() = %s, want %s", result, tc.want)
			}
		})
	}
}

func TestGetStreamPath(t *testing.T) {
	cases := map[string]struct {
		input absto.Item
		want  string
	}{
		"simple": {
			absto.Item{
				ID:       "dd29ecf524b030a65261e3059c48ab9e1ecb2585",
				Pathname: "/path/to/file.mov",
			},
			"/.fibr/path/to/dd29ecf524b030a65261e3059c48ab9e1ecb2585.m3u8",
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			if result := getStreamPath(tc.input); result != tc.want {
				t.Errorf("getStreamPath() = %s, want %s", result, tc.want)
			}
		})
	}
}
