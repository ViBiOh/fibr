package fibr

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsMethodAllowed(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input *http.Request
		want  bool
	}{
		"valid": {
			httptest.NewRequest(http.MethodGet, "/", nil),
			true,
		},
		"invalid": {
			httptest.NewRequest(http.MethodOptions, "/", nil),
			false,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if result := isMethodAllowed(tc.input); result != tc.want {
				t.Errorf("isMethodAllowed(%#v) = %#v, want %#v", tc.input, result, tc.want)
			}
		})
	}
}
