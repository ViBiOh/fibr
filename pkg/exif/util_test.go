package exif

import (
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func TestGetExifPath(t *testing.T) {
	type args struct {
		item provider.StorageItem
	}

	cases := []struct {
		intention string
		args      args
		want      string
	}{
		{
			"simple",
			args{
				item: provider.StorageItem{
					Pathname: "/photos/image.jpeg",
				},
			},
			"/.fibr/photos/dd29ecf524b030a65261e3059c48ab9e1ecb2585.json",
		},
		{
			"simple",
			args{
				item: provider.StorageItem{
					Pathname: "/photos",
					IsDir:    true,
				},
			},
			"/.fibr/photos/aggregate.json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := getExifPath(tc.args.item); got != tc.want {
				t.Errorf("getExifPath() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}
