package thumbnail

import (
	"testing"

	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/golang/mock/gomock"
)

func TestCanHaveThumbnail(t *testing.T) {
	cases := []struct {
		intention string
		instance  App
		input     provider.StorageItem
		want      bool
	}{
		{
			"empty",
			App{},
			provider.StorageItem{},
			false,
		},
		{
			"image",
			App{},
			provider.StorageItem{
				Name: "test.png",
			},
			true,
		},
		{
			"pdf",
			App{},
			provider.StorageItem{
				Name: "test.pdf",
			},
			true,
		},
		{
			"video",
			App{},
			provider.StorageItem{
				Name: "test.avi",
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
		input     provider.StorageItem
		want      bool
	}{
		{
			"not found",
			App{},
			provider.StorageItem{
				Pathname: "path/to/error",
				IsDir:    true,
			},
			false,
		},
		{
			"found",
			App{},
			provider.StorageItem{
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
				storageMock.EXPECT().Info(gomock.Any()).Return(provider.StorageItem{}, nil)
			}

			if result := tc.instance.HasThumbnail(tc.input); result != tc.want {
				t.Errorf("HasThumbnail() = %t, want %t", result, tc.want)
			}
		})
	}
}

func TestGetThumbnailPath(t *testing.T) {
	cases := []struct {
		intention string
		input     provider.StorageItem
		want      string
	}{
		{
			"simple",
			provider.StorageItem{
				Pathname: "/path/to/file.png",
			},
			".fibr/path/to/file.webp",
		},
		{
			"directory",
			provider.StorageItem{
				Pathname: "/path/to/file/",
				IsDir:    true,
			},
			".fibr/path/to/file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if result := getThumbnailPath(tc.input); result != tc.want {
				t.Errorf("getThumbnailPath() = %s, want %s", result, tc.want)
			}
		})
	}
}
