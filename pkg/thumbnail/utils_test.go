package thumbnail

import (
	"context"
	"testing"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/golang/mock/gomock"
)

func TestCanHaveThumbnail(t *testing.T) {
	cases := []struct {
		intention string
		instance  App
		input     absto.Item
		want      bool
	}{
		{
			"empty",
			App{},
			absto.Item{},
			false,
		},
		{
			"image",
			App{},
			absto.Item{
				Name:      "test.png",
				Extension: ".png",
			},
			true,
		},
		{
			"pdf",
			App{},
			absto.Item{
				Name:      "test.pdf",
				Extension: ".pdf",
			},
			true,
		},
		{
			"video",
			App{},
			absto.Item{
				Name:      "test.avi",
				Extension: ".avi",
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if result := tc.instance.CanHaveThumbnail(tc.input); result != tc.want {
				t.Errorf("CanHaveThumbnail() = %t, want %t", result, tc.want)
			}
		})
	}
}

func TestHasThumbnail(t *testing.T) {
	cases := []struct {
		intention string
		instance  App
		input     absto.Item
		want      bool
	}{
		{
			"not found",
			App{},
			absto.Item{
				Pathname: "path/to/error",
				IsDir:    true,
			},
			false,
		},
		{
			"found",
			App{},
			absto.Item{
				Pathname: "path/to/valid",
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			storageMock := mocks.NewStorage(ctrl)

			tc.instance.storageApp = storageMock

			if tc.intention == "found" {
				storageMock.EXPECT().Info(gomock.Any(), gomock.Any()).Return(absto.Item{}, nil)
			}

			if result := tc.instance.HasThumbnail(context.Background(), tc.input, SmallSize); result != tc.want {
				t.Errorf("HasThumbnail() = %t, want %t", result, tc.want)
			}
		})
	}
}

func TestPathForScale(t *testing.T) {
	cases := []struct {
		intention string
		instance  App
		input     absto.Item
		want      string
	}{
		{
			"simple",
			App{},
			absto.Item{
				ID:       "dd29ecf524b030a65261e3059c48ab9e1ecb2585",
				Pathname: "/path/to/file.png",
			},
			"/.fibr/path/to/dd29ecf524b030a65261e3059c48ab9e1ecb2585.webp",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if result := tc.instance.PathForScale(tc.input, SmallSize); result != tc.want {
				t.Errorf("PathForScale() = %s, want %s", result, tc.want)
			}
		})
	}
}

func TestGetStreamPath(t *testing.T) {
	cases := []struct {
		intention string
		input     absto.Item
		want      string
	}{
		{
			"simple",
			absto.Item{
				ID:       "dd29ecf524b030a65261e3059c48ab9e1ecb2585",
				Pathname: "/path/to/file.mov",
			},
			"/.fibr/path/to/dd29ecf524b030a65261e3059c48ab9e1ecb2585.m3u8",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if result := getStreamPath(tc.input); result != tc.want {
				t.Errorf("getStreamPath() = %s, want %s", result, tc.want)
			}
		})
	}
}
