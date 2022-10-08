package share

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/clock"
)

func TestPurgeExpiredShares(t *testing.T) {
	cases := map[string]struct {
		instance *App
		want     map[string]provider.Share
	}{
		"empty": {
			&App{
				clock:  clock.New(time.Date(2021, 0o5, 0o1, 14, 0o0, 0o0, 0, time.UTC)),
				shares: make(map[string]provider.Share),
			},
			make(map[string]provider.Share),
		},
		"purge at boundaries": {
			&App{
				clock: clock.New(time.Date(2021, 0o5, 0o1, 14, 0o0, 0o0, 0, time.UTC)),
				shares: map[string]provider.Share{
					"1": {
						ID:       "1",
						Creation: time.Date(2021, 0o5, 0o1, 12, 0o0, 0o0, 0, time.UTC),
						Duration: time.Hour,
					},
					"2": {
						ID:       "2",
						Creation: time.Date(2021, 0o5, 0o1, 12, 0o0, 0o0, 0, time.UTC),
						Duration: time.Hour * 24,
					},
					"22": {
						ID:       "22",
						Creation: time.Date(2021, 0o5, 0o1, 12, 0o0, 0o0, 0, time.UTC),
						Duration: 0,
					},
					"3": {
						ID:       "3",
						Creation: time.Date(2021, 0o5, 0o1, 12, 0o0, 0o0, 0, time.UTC),
						Duration: time.Hour,
					},
				},
			},
			map[string]provider.Share{
				"2": {
					ID:       "2",
					Creation: time.Date(2021, 0o5, 0o1, 12, 0o0, 0o0, 0, time.UTC),
					Duration: time.Hour * 24,
				},
				"22": {
					ID:       "22",
					Creation: time.Date(2021, 0o5, 0o1, 12, 0o0, 0o0, 0, time.UTC),
					Duration: 0,
				},
			},
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			tc.instance.purgeExpiredShares(context.TODO())
			got := tc.instance.shares

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("purgeExpiredShares() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
