package fibr

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsMethodAllowed(t *testing.T) {
	cases := []struct {
		intention string
		input     *http.Request
		want      bool
	}{
		{
			"valid",
			httptest.NewRequest(http.MethodGet, "/", nil),
			true,
		},
		{
			"invalid",
			httptest.NewRequest(http.MethodOptions, "/", nil),
			false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := isMethodAllowed(testCase.input); result != testCase.want {
				t.Errorf("isMethodAllowed(%#v) = %#v, want %#v", testCase.input, result, testCase.want)
			}
		})
	}
}
