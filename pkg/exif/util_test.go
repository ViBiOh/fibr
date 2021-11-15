package exif

import (
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func TestGetExifPath(t *testing.T) {
	type args struct {
		item   provider.StorageItem
		suffix string
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
				suffix: exifMetadataFilename,
			},
			".fibr/photos/image.json",
		},
		{
			"simple",
			args{
				item: provider.StorageItem{
					Pathname: "/photos",
					IsDir:    true,
				},
				suffix: aggregateMetadataFilename,
			},
			".fibr/photos/aggregate.json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := getExifPath(tc.args.item, tc.args.suffix); got != tc.want {
				t.Errorf("getExifPath() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}
