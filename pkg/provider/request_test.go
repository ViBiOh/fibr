package provider

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ViBiOh/httputils/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

func TestCheckPassword(t *testing.T) {
	password, err := bcrypt.GenerateFromPassword([]byte("test"), bcrypt.DefaultCost)
	if err != nil {
		t.Errorf("unable to create bcrypted password: %#v", err)
	}

	invalidAuth := httptest.NewRequest(http.MethodGet, "/", nil)
	invalidAuth.Header.Set("Authorization", "invalid")

	invalidFormat := httptest.NewRequest(http.MethodGet, "/", nil)
	invalidFormat.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("test"))))

	invalidPassword := httptest.NewRequest(http.MethodGet, "/", nil)
	invalidPassword.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("test:password"))))

	valid := httptest.NewRequest(http.MethodGet, "/", nil)
	valid.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("test:test"))))

	var cases = []struct {
		intention string
		share     *Share
		request   *http.Request
		want      error
	}{
		{
			"no password",
			&Share{},
			httptest.NewRequest(http.MethodGet, "/", nil),
			nil,
		},
		{
			"password no auth",
			&Share{
				Password: string(password),
			},
			httptest.NewRequest(http.MethodGet, "/", nil),
			errors.New("empty authorization header"),
		},
		{
			"invalid authorization",
			&Share{
				Password: string(password),
			},
			invalidAuth,
			errors.New("illegal base64 data at input byte 4"),
		},
		{
			"invalid format",
			&Share{
				Password: string(password),
			},
			invalidFormat,
			errors.New("invalid format for basic auth"),
		},
		{
			"invalid password",
			&Share{
				Password: string(password),
			},
			invalidPassword,
			errors.New("invalid credentials"),
		},
		{
			"valid",
			&Share{
				Password: string(password),
			},
			valid,
			nil,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			err := testCase.share.CheckPassword(testCase.request)

			failed := false

			if err == nil && testCase.want != nil {
				failed = true
			} else if err != nil && testCase.want == nil {
				failed = true
			} else if err != nil && err.Error() != testCase.want.Error() {
				failed = true
			}

			if failed {
				t.Errorf("%#v.CheckPassword(%#v) = %#v, want %#v", testCase.share, testCase.request, err, testCase.want)
			}
		})
	}
}
