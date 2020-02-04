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
