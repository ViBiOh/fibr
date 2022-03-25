package exif

import (
	"testing"

	absto "github.com/ViBiOh/absto/pkg/model"
)

func TestGetExifPath(t *testing.T) {
	type args struct {
		item absto.Item
	}

	cases := map[string]struct {
		args args
		want string
	}{
		"simple": {
			args{
				item: absto.Item{
					ID:       "dd29ecf524b030a65261e3059c48ab9e1ecb2585",
					Pathname: "/photos/image.jpeg",
				},
			},
			"/.fibr/photos/dd29ecf524b030a65261e3059c48ab9e1ecb2585.json",
		},
		"dir": {
			args{
				item: absto.Item{
					Pathname: "/photos",
					IsDir:    true,
				},
			},
			"/.fibr/photos/aggregate.json",
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			if got := Path(tc.args.item); got != tc.want {
				t.Errorf("getExifPath() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}
