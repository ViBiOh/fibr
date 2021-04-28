package crud

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func TestPurgeExpiredMetadatas(t *testing.T) {
	var cases = []struct {
		intention string
		current   []*provider.Share
		want      []*provider.Share
	}{
		{
			"empty",
			nil,
			nil,
		},
		{
			"purge at boundaries",
			[]*provider.Share{
				{
					ID:       "1",
					Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
					Duration: time.Hour,
				},
				{
					ID:       "2",
					Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
					Duration: time.Hour * 24,
				},
				{
					ID:       "22",
					Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
					Duration: 0,
				},
				{
					ID:       "3",
					Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
					Duration: time.Hour,
				},
			},
			[]*provider.Share{
				{
					ID:       "2",
					Creation: time.Date(2021, 05, 01, 12, 00, 00, 0, time.UTC),
					Duration: time.Hour * 24,
				},
				{
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
				metadataLock:    sync.Mutex{},
				metadatas:       tc.current,
				clock: Clock{
					now: time.Date(2021, 05, 01, 14, 00, 00, 0, time.UTC),
				},
			}

			instance.purgeExpiredMetadatas()

			if !reflect.DeepEqual(instance.metadatas, tc.want) {
				t.Errorf("purgeExpiredMetadatas() = %+v, want %+v", instance.metadatas, tc.want)
			}
		})
	}
}
