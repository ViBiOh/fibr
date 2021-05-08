package metadata

import (
	"reflect"
	"testing"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func TestPurgeExpiredMetadatas(t *testing.T) {
	var cases = []struct {
		intention string
		instance  *app
		want      map[string]provider.Share
	}{
		{
			"empty",
			&app{
				clock: &Clock{
					now: time.Date(2021, 05, 01, 14, 00, 00, 0, time.UTC),
				},
				metadatas: make(map[string]provider.Share),
			},
			make(map[string]provider.Share),
		},
		{
			"purge at boundaries",
			&app{
				clock: &Clock{
					now: time.Date(2021, 05, 01, 14, 00, 00, 0, time.UTC),
				},
				metadatas: map[string]provider.Share{
					"1": {
						ID:       "1",
						Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
						Duration: time.Hour,
					},
					"2": {
						ID:       "2",
						Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
						Duration: time.Hour * 24,
					},
					"22": {
						ID:       "22",
						Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
						Duration: 0,
					},
					"3": {
						ID:       "3",
						Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
						Duration: time.Hour,
					},
				},
			},
			map[string]provider.Share{
				"2": {
					ID:       "2",
					Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
					Duration: time.Hour * 24,
				},
				"22": {
					ID:       "22",
					Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
					Duration: 0,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			tc.instance.purgeExpiredMetadatas()
			got := tc.instance.metadatas

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("purgeExpiredMetadatas() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
