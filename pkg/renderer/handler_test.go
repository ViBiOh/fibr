package renderer

import (
	"reflect"
	"testing"
)

func TestFindIndex(t *testing.T) {
	type args struct {
		arr   []string
		value string
	}

	var cases = []struct {
		intention string
		args      args
		want      int
	}{
		{
			"empty",
			args{},
			-1,
		},
		{
			"single element",
			args{
				arr:   []string{"localhost"},
				value: "localhost",
			},
			0,
		},
		{
			"multiple element",
			args{
				arr:   []string{"localhost", "::1", "world.com"},
				value: "::1",
			},
			1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := findIndex(tc.args.arr, tc.args.value); got != tc.want {
				t.Errorf("findIndex() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestRemoveIndex(t *testing.T) {
	type args struct {
		arr   []string
		index int
	}

	var cases = []struct {
		intention string
		args      args
		want      []string
	}{
		{
			"empty",
			args{},
			nil,
		},
		{
			"negative",
			args{
				arr:   []string{"localhost"},
				index: -1,
			},
			[]string{"localhost"},
		},
		{
			"index out of range",
			args{
				arr:   []string{"localhost"},
				index: 1,
			},
			[]string{"localhost"},
		},
		{
			"valid",
			args{
				arr:   []string{"localhost"},
				index: 0,
			},
			[]string{},
		},
		{
			"multiple",
			args{
				arr:   []string{"localhost", "::1", "world.com"},
				index: 1,
			},
			[]string{"localhost", "world.com"},
		},
		{
			"upper bounds",
			args{
				arr:   []string{"localhost", "::1", "world.com"},
				index: 2,
			},
			[]string{"localhost", "::1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := removeIndex(tc.args.arr, tc.args.index); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("removeIndex() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
