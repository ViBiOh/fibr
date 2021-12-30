package crud

import (
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func TestMatchSize(t *testing.T) {
	type args struct {
		item        provider.StorageItem
		size        int64
		greaterThan bool
	}

	cases := []struct {
		intention string
		instance  search
		args      args
		want      bool
	}{
		{
			"no size",
			search{
				Size:        0,
				GreaterThan: true,
			},
			args{
				item: provider.StorageItem{Size: 1000},
			},
			true,
		},
		{
			"greater for greater",
			search{
				Size:        900,
				GreaterThan: true,
			},
			args{
				item: provider.StorageItem{Size: 1000},
			},
			true,
		},
		{
			"greater for lower",
			search{
				Size:        900,
				GreaterThan: false,
			},
			args{
				item: provider.StorageItem{Size: 1000},
			},
			false,
		},
		{
			"lower for lower",
			search{
				Size:        900,
				GreaterThan: false,
			},
			args{
				item: provider.StorageItem{Size: 800},
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := tc.instance.matchSize(tc.args.item); got != tc.want {
				t.Errorf("MatchSize() = %t, want %t", got, tc.want)
			}
		})
	}
}
