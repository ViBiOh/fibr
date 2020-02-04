package thumbnail

import (
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func TestCanHaveThumbnail(t *testing.T) {
	var cases = []struct {
		intention string
		input     provider.StorageItem
		want      bool
	}{
		{
			"empty",
			provider.StorageItem{},
			false,
		},
		{
			"image",
			provider.StorageItem{
				Name: "test.png",
			},
			true,
		},
		{
			"pdf",
			provider.StorageItem{
				Name: "test.pdf",
			},
			true,
		},
		{
			"video",
			provider.StorageItem{
				Name: "test.avi",
			},
			true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := CanHaveThumbnail(testCase.input); result != testCase.want {
				t.Errorf("CanHaveThumbnail() = %t, want %t", result, testCase.want)
			}
		})
	}
}

func TestHasThumbnail(t *testing.T) {
	var cases = []struct {
		intention string
		instance  app
		input     provider.StorageItem
		want      bool
	}{
		{
			"not enabled",
			app{},
			provider.StorageItem{},
			false,
		},
		{
			"not found",
			app{
				storage:  stubStorage{},
				imageURL: "http://localhost",
				videoURL: "http://localhost",
			},
			provider.StorageItem{
				Pathname: "path/to/error",
				IsDir:    true,
			},
			false,
		},
		{
			"found",
			app{
				storage:  stubStorage{},
				imageURL: "http://localhost",
				videoURL: "http://localhost",
			},
			provider.StorageItem{
				Pathname: "path/to/valid",
			},
			true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := testCase.instance.HasThumbnail(testCase.input); result != testCase.want {
				t.Errorf("HasThumbnail() = %t, want %t", result, testCase.want)
			}
		})
	}
}

func TestGetThumbnailPath(t *testing.T) {
	var cases = []struct {
		intention string
		input     provider.StorageItem
		want      string
	}{
		{
			"simple",
			provider.StorageItem{
				Pathname: "/path/to/file.png",
			},
			".fibr/path/to/file.jpg",
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

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := getThumbnailPath(testCase.input); result != testCase.want {
				t.Errorf("getThumbnailPath() = %s, want %s", result, testCase.want)
			}
		})
	}
}
