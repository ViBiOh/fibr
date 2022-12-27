package search

import (
	"testing"
	"time"
)

func TestComputeSince(t *testing.T) {
	t.Parallel()

	type args struct {
		input time.Time
		unit  string
		value int
	}

	cases := map[string]struct {
		args args
		want time.Time
	}{
		"unknown": {
			args{
				input: time.Date(2004, 2, 29, 12, 0, 0, 0, time.UTC),
			},
			time.Date(2004, 2, 29, 12, 0, 0, 0, time.UTC),
		},
		"days": {
			args{
				input: time.Date(2004, 2, 29, 12, 0, 0, 0, time.UTC),
				unit:  "days",
				value: 10,
			},
			time.Date(2004, 2, 19, 12, 0, 0, 0, time.UTC),
		},
		"months": {
			args{
				input: time.Date(2004, 2, 29, 12, 0, 0, 0, time.UTC),
				unit:  "months",
				value: 1,
			},
			time.Date(2004, 1, 29, 12, 0, 0, 0, time.UTC),
		},
		"months more day": {
			args{
				input: time.Date(2004, 10, 31, 12, 0, 0, 0, time.UTC),
				unit:  "months",
				value: 1,
			},
			time.Date(2004, 9, 30, 12, 0, 0, 0, time.UTC),
		},
		"months february": {
			args{
				input: time.Date(2004, 3, 31, 12, 0, 0, 0, time.UTC),
				unit:  "months",
				value: 1,
			},
			time.Date(2004, 2, 29, 12, 0, 0, 0, time.UTC),
		},
		"many months": {
			args{
				input: time.Date(2004, 12, 12, 12, 0, 0, 0, time.UTC),
				unit:  "months",
				value: 12,
			},
			time.Date(2003, 12, 12, 12, 0, 0, 0, time.UTC),
		},
		"leap years": {
			args{
				input: time.Date(2004, 2, 29, 12, 0, 0, 0, time.UTC),
				unit:  "years",
				value: 1,
			},
			time.Date(2003, 2, 28, 12, 0, 0, 0, time.UTC),
		},
		"years": {
			args{
				input: time.Date(2007, 2, 28, 12, 0, 0, 0, time.UTC),
				unit:  "years",
				value: 3,
			},
			time.Date(2004, 2, 28, 12, 0, 0, 0, time.UTC),
		},
	}

	for intention, testCase := range cases {
		intention, testCase := intention, testCase

		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if got := computeSince(testCase.args.input, testCase.args.unit, testCase.args.value); got != testCase.want {
				t.Errorf("computeSince() = %s, want %s", got, testCase.want)
			}
		})
	}
}
