package fibr

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

func TestIsMethodAllowed(t *testing.T) {
	var cases = []struct {
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

func TestCheckSharePassword(t *testing.T) {
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
		request   *http.Request
		share     *provider.Share
		want      error
	}{
		{
			"no password",
			httptest.NewRequest(http.MethodGet, "/", nil),
			&provider.Share{},
			nil,
		},
		{
			"password no auth",
			httptest.NewRequest(http.MethodGet, "/", nil),
			&provider.Share{
				Password: string(password),
			},
			errEmptyAuthorizationHeader,
		},
		{
			"invalid authorization",
			invalidAuth,
			&provider.Share{
				Password: string(password),
			},
			errors.New("illegal base64 data at input byte 4"),
		},
		{
			"invalid format",
			invalidFormat,
			&provider.Share{
				Password: string(password),
			},
			errors.New("invalid format for basic auth"),
		},
		{
			"invalid password",
			invalidPassword,
			&provider.Share{
				Password: string(password),
			},
			errors.New("invalid credentials"),
		},
		{
			"valid",
			valid,
			&provider.Share{
				Password: string(password),
			},
			nil,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			err := checkSharePassword(testCase.request, testCase.share)

			failed := false

			if err == nil && testCase.want != nil {
				failed = true
			} else if err != nil && testCase.want == nil {
				failed = true
			} else if err != nil && err.Error() != testCase.want.Error() {
				failed = true
			}

			if failed {
				t.Errorf("checkSharePassword(%#v, %#v) = %#v, want %#v", testCase.request, testCase.share, err, testCase.want)
			}
		})
	}
}
