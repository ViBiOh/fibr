package crud

import (
	"testing"

	absto "github.com/ViBiOh/absto/pkg/model"
)

func TestMatchSize(t *testing.T) {
	type args struct {
		item absto.Item
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
				size:        0,
				greaterThan: true,
			},
			args{
				item: absto.Item{Size: 1000},
			},
			true,
		},
		{
			"greater for greater",
			search{
				size:        900,
				greaterThan: true,
			},
			args{
				item: absto.Item{Size: 1000},
			},
			true,
		},
		{
			"greater for lower",
			search{
				size:        900,
				greaterThan: false,
			},
			args{
				item: absto.Item{Size: 1000},
			},
			false,
		},
		{
			"lower for lower",
			search{
				size:        900,
				greaterThan: false,
			},
			args{
				item: absto.Item{Size: 800},
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
