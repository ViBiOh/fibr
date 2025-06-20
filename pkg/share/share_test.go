package share

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	"go.uber.org/mock/gomock"
)

func TestPurgeExpiredShares(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		instance *Service
		want     map[string]provider.Share
	}{
		"empty": {
			&Service{
				clock:  func() time.Time { return time.Date(2021, 0o5, 0o1, 14, 0o0, 0o0, 0, time.UTC) },
				shares: make(map[string]provider.Share),
			},
			make(map[string]provider.Share),
		},
		"purge at boundaries": {
			&Service{
				clock: func() time.Time { return time.Date(2021, 0o5, 0o1, 14, 0o0, 0o0, 0, time.UTC) },
				shares: map[string]provider.Share{
					"1": {
						ID:       "1",
						Created:  time.Date(2021, 0o5, 0o1, 12, 0o0, 0o0, 0, time.UTC),
						Duration: time.Hour,
					},
					"2": {
						ID:       "2",
						Created:  time.Date(2021, 0o5, 0o1, 12, 0o0, 0o0, 0, time.UTC),
						Duration: time.Hour * 24,
					},
					"22": {
						ID:       "22",
						Created:  time.Date(2021, 0o5, 0o1, 12, 0o0, 0o0, 0, time.UTC),
						Duration: 0,
					},
					"3": {
						ID:       "3",
						Created:  time.Date(2021, 0o5, 0o1, 12, 0o0, 0o0, 0, time.UTC),
						Duration: time.Hour,
					},
				},
			},
			map[string]provider.Share{
				"2": {
					ID:       "2",
					Created:  time.Date(2021, 0o5, 0o1, 12, 0o0, 0o0, 0, time.UTC),
					Duration: time.Hour * 24,
				},
				"22": {
					ID:       "22",
					Created:  time.Date(2021, 0o5, 0o1, 12, 0o0, 0o0, 0, time.UTC),
					Duration: 0,
				},
			},
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			redisMocks := mocks.NewRedisClient(ctrl)

			testCase.instance.redisClient = redisMocks

			switch intention {
			case "purge at boundaries":
				redisMocks.EXPECT().PublishJSON(gomock.Any(), gomock.Any(), provider.Share{ID: "1"})
				redisMocks.EXPECT().PublishJSON(gomock.Any(), gomock.Any(), provider.Share{ID: "3"})
			}

			testCase.instance.purgeExpiredShares(context.TODO())
			got := testCase.instance.shares

			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("purgeExpiredShares() = %+v, want %+v", got, testCase.want)
			}
		})
	}
}
