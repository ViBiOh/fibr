package exif

import (
	"errors"
	"strings"
	"testing"
)

func TestConvertDegreeMinuteSecondToDecimal(t *testing.T) {
	type args struct {
		location string
	}

	var cases = []struct {
		intention string
		args      args
		want      string
		wantErr   error
	}{
		{
			"empty",
			args{
				location: "",
			},
			"",
			errors.New("unable to parse GPS data"),
		},
		{
			"north",
			args{
				location: "1 deg 2' 3\" N",
			},
			"1.034167",
			nil,
		},
		{
			"west",
			args{
				location: "1 deg 2' 3\" W",
			},
			"-1.034167",
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			got, gotErr := convertDegreeMinuteSecondToDecimal(tc.args.location)

			failed := false

			if tc.wantErr == nil && gotErr != nil {
				failed = true
			} else if tc.wantErr != nil && gotErr == nil {
				failed = true
			} else if tc.wantErr != nil && !strings.Contains(gotErr.Error(), tc.wantErr.Error()) {
				failed = true
			} else if got != tc.want {
				failed = true
			}

			if failed {
				t.Errorf("convertDegreeMinuteSecondToDecimal() = (`%s`, `%s`), want (`%s`, `%s`)", got, gotErr, tc.want, tc.wantErr)
			}
		})
	}
}
