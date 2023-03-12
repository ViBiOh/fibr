package provider

import (
	"io"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	cases := map[string]struct {
		name        string
		removeSlash bool
		want        string
		wantErr     error
	}{
		"should work with empty name": {
			"",
			true,
			"",
			nil,
		},
		"should replace space by underscore": {
			"fibr is a file browser",
			true,
			"fibr_is_a_file_browser",
			nil,
		},
		"should replace diacritics and special chars": {
			`L'Œil "où", l'ouïe fine au Ø`,
			true,
			"l_oeil_ou_l_ouie_fine_au_oe",
			nil,
		},
		"should not replace slash if not asked": {
			"path/name",
			false,
			"path/name",
			nil,
		},
		"should replace slash if asked": {
			"path/name",
			true,
			"path_name",
			nil,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			result, err := SanitizeName(tc.name, tc.removeSlash)

			failed := false

			if err == nil && tc.wantErr != nil {
				failed = true
			} else if err != nil && tc.wantErr == nil {
				failed = true
			} else if err != nil && err.Error() != tc.wantErr.Error() {
				failed = true
			} else if result != tc.want {
				failed = true
			}

			if failed {
				t.Errorf("SanitizeName() = (%#v, `%s`), want (%#v, `%s`)", result, err, tc.want, tc.wantErr)
			}
		})
	}
}

func TestSafeWrite(t *testing.T) {
	type args struct {
		writer  io.Writer
		content string
	}

	cases := map[string]struct {
		args args
	}{
		"no panic": {
			args{
				writer:  io.Discard,
				content: "test",
			},
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			SafeWrite(tc.args.writer, tc.args.content)
		})
	}
}
