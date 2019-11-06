package provider

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestGetFilepath(t *testing.T) {
	var cases = []struct {
		intention string
		request   Request
		input     string
		want      string
	}{
		{
			"simple",
			Request{
				Path: "index",
			},
			"",
			"index",
		},
		{
			"with given path",
			Request{
				Path: "index",
			},
			"root.html",
			"index/root.html",
		},
		{
			"with share",
			Request{
				Path: "index",
				Share: &Share{
					Path: "/shared/",
				},
			},
			"root.html",
			"/shared/index/root.html",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := testCase.request.GetFilepath(testCase.input); result != testCase.want {
				t.Errorf("%#v.GetFilepath(`%s`) = `%s`, want `%s`", testCase.request, testCase.input, result, testCase.want)
			}
		})
	}
}

func TestGetURI(t *testing.T) {
	var cases = []struct {
		intention string
		request   Request
		input     string
		want      string
	}{
		{
			"simple",
			Request{
				Path: "index",
			},
			"",
			"index",
		},
		{
			"with given path",
			Request{
				Path: "index",
			},
			"root.html",
			"index/root.html",
		},
		{
			"with share",
			Request{
				Path: "index",
				Share: &Share{
					ID: "abcd1234",
				},
			},
			"root.html",
			"abcd1234/index/root.html",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := testCase.request.GetURI(testCase.input); result != testCase.want {
				t.Errorf("%#v.GetFilepath(`%s`) = `%s`, want `%s`", testCase.request, testCase.input, result, testCase.want)
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	password, err := bcrypt.GenerateFromPassword([]byte("test"), bcrypt.DefaultCost)
	if err != nil {
		t.Errorf("unable to create bcrypted password: %s", err)
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

func TestNewError(t *testing.T) {
	var cases = []struct {
		intention string
		status    int
		err       error
		want      *Error
	}{
		{
			"empty",
			0,
			nil,
			nil,
		},
		{
			"given error",
			500,
			errors.New("invalid value"),
			&Error{
				Status: 500,
				Err:    errors.New("invalid value"),
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := NewError(testCase.status, testCase.err); !reflect.DeepEqual(result, testCase.want) {
				t.Errorf("NewError(%d, %#v) = %#v, want %#v", testCase.status, testCase.err, result, testCase.want)
			}
		})
	}
}

func TestExtension(t *testing.T) {
	var cases = []struct {
		intention string
		input     StorageItem
		want      string
	}{
		{
			"simple",
			StorageItem{
				Name: "test.TXT",
			},
			".txt",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := testCase.input.Extension(); result != testCase.want {
				t.Errorf("Extension() = `%s`, want `%s`", result, testCase.want)
			}
		})
	}
}

func TestMime(t *testing.T) {
	var cases = []struct {
		intention string
		input     StorageItem
		want      string
	}{
		{
			"empty",
			StorageItem{
				Name: "test",
			},
			"",
		},
		{
			"simple",
			StorageItem{
				Name: "test.TXT",
			},
			"text/plain; charset=utf-8",
		},
		{
			"golang",
			StorageItem{
				Name: "main.go",
			},
			"text/plain; charset=utf-8",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := testCase.input.Mime(); result != testCase.want {
				t.Errorf("Mime() = `%s`, want `%s`", result, testCase.want)
			}
		})
	}
}

func TestIsImage(t *testing.T) {
	var cases = []struct {
		intention string
		input     StorageItem
		want      bool
	}{
		{
			"simple",
			StorageItem{
				Name: "test.png",
			},
			true,
		},
		{
			"raw image",
			StorageItem{
				Name: "test.raw",
			},
			false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := testCase.input.IsImage(); result != testCase.want {
				t.Errorf("IsImage() = `%v`, want `%v`", result, testCase.want)
			}
		})
	}
}

func TestIsVideo(t *testing.T) {
	var cases = []struct {
		intention string
		input     StorageItem
		want      bool
	}{
		{
			"simple",
			StorageItem{
				Name: "test.mov",
			},
			true,
		},
		{
			"old video",
			StorageItem{
				Name: "test.divx",
			},
			false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := testCase.input.IsVideo(); result != testCase.want {
				t.Errorf("IsVideo() = `%v`, want `%v`", result, testCase.want)
			}
		})
	}
}
