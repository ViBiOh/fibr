package crud

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func TestPurgeExpiredMetadatas(t *testing.T) {
	var boundaries sync.Map
	boundaries.Store("1", provider.Share{
		ID:       "1",
		Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
		Duration: time.Hour,
	})
	boundaries.Store("2", provider.Share{
		ID:       "2",
		Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
		Duration: time.Hour * 24,
	})
	boundaries.Store("22", provider.Share{
		ID:       "22",
		Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
		Duration: 0,
	})
	boundaries.Store("3", provider.Share{
		ID:       "3",
		Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
		Duration: time.Hour,
	})

	var filtered sync.Map
	filtered.Store("2", provider.Share{
		ID:       "2",
		Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
		Duration: time.Hour * 24,
	})
	filtered.Store("22", provider.Share{
		ID:       "22",
		Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
		Duration: 0,
	})

	var cases = []struct {
		intention string
		current   map[string]provider.Share
		want      map[string]provider.Share
	}{
		{
			"empty",
			make(map[string]provider.Share, 0),
			make(map[string]provider.Share, 0),
		},
		{
			"purge at boundaries",
			map[string]provider.Share{
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
			instance := &app{
				metadataEnabled: true,
				clock: &Clock{
					now: time.Date(2021, 05, 01, 14, 00, 00, 0, time.UTC),
				},
			}

			for key, value := range tc.current {
				instance.metadatas.Store(key, value)
			}

			instance.purgeExpiredMetadatas()
			got := instance.dumpMetadatas()

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("purgeExpiredMetadatas() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
