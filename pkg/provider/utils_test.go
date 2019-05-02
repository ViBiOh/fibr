package provider

import "testing"

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
			`Le terme "où", l'ouïe fine`,
			true,
			"le_terme_ou,_louie_fine",
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

	var failed bool

	for _, testCase := range cases {
		result, err := SanitizeName(testCase.name, testCase.removeSlash)

		failed = false

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
			t.Errorf("%s\nSanitizeName(`%s`) = (%+v, %+v), want (%+v, %+v)", testCase.intention, testCase.name, result, err, testCase.want, testCase.wantErr)
		}
	}
}
