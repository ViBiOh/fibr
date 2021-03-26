package provider

import (
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	var cases = []struct {
		intention   string
		name        string
		removeSlash bool
		want        string
		wantErr     error
	}{
		{
			"should work with empty name",
			"",
			true,
			"",
			nil,
		},
		{
			"should replace space by underscore",
			"fibr is a file browser",
			true,
			"fibr_is_a_file_browser",
			nil,
		},
		{
			"should replace diacritics and special chars",
			`L'Œil "où", l'ouïe fine au Ø`,
			true,
			"l_oeil_ou_l_ouie_fine_au_oe",
			nil,
		},
		{
			"should not replace slash if not asked",
			"path/name",
			false,
			"path/name",
			nil,
		},
		{
			"should replace slash if asked",
			"path/name",
			true,
			"path_name",
			nil,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			result, err := SanitizeName(testCase.name, testCase.removeSlash)

			failed := false

			if err == nil && testCase.wantErr != nil {
				failed = true
			} else if err != nil && testCase.wantErr == nil {
				failed = true
			} else if err != nil && err.Error() != testCase.wantErr.Error() {
				failed = true
			} else if result != testCase.want {
				failed = true
			}

			if failed {
				t.Errorf("SanitizeName() = (%#v, `%s`), want (%#v, `%s`)", result, err, testCase.want, testCase.wantErr)
			}
		})
	}
}

func TestSafeWrite(t *testing.T) {
	type args struct {
		writer  io.Writer
		content string
	}

	var cases = []struct {
		intention string
		args      args
	}{
		{
			"no panic",
			args{
				writer:  io.Discard,
				content: "test",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			SafeWrite(tc.args.writer, tc.args.content)
		})
	}
}

func TestIsNotExist(t *testing.T) {
	var cases = []struct {
		intention string
		input     error
		want      bool
	}{
		{
			"nil error",
			nil,
			false,
		},
		{
			"empty error",
			errors.New("invalid value"),
			false,
		},
		{
			"file error",
			ErrNotExist(errors.New("index.html: no such file or directory")),
			true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := IsNotExist(testCase.input); result != testCase.want {
				t.Errorf("IsNotExist() = %#v, want %#v", result, testCase.want)
			}
		})
	}
}

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
			if got := FindIndex(tc.args.arr, tc.args.value); got != tc.want {
				t.Errorf("FindIndex() = %d, want %d", got, tc.want)
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
			if got := RemoveIndex(tc.args.arr, tc.args.index); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("RemoveIndex() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
