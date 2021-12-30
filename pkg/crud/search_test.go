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
		args      args
		want      bool
	}{
		{
			"no size",
			args{
				item:        provider.StorageItem{Size: 1000},
				size:        0,
				greaterThan: true,
			},
			true,
		},
		{
			"greater for greater",
			args{
				item:        provider.StorageItem{Size: 1000},
				size:        900,
				greaterThan: true,
			},
			true,
		},
		{
			"greater for lower",
			args{
				item:        provider.StorageItem{Size: 1000},
				size:        900,
				greaterThan: false,
			},
			false,
		},
		{
			"lower for lower",
			args{
				item:        provider.StorageItem{Size: 800},
				size:        900,
				greaterThan: false,
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := matchSize(tc.args.item, tc.args.size, tc.args.greaterThan); got != tc.want {
				t.Errorf("MatchSize() = %t, want %t", got, tc.want)
			}
		})
	}
}
