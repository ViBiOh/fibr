package provider

import (
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestCheckPassword(t *testing.T) {
	password, err := bcrypt.GenerateFromPassword([]byte("test"), bcrypt.DefaultCost)
	if err != nil {
		t.Errorf("create bcrypted password: %s", err)
	}

	cases := map[string]struct {
		share  Share
		header string
		want   error
	}{
		"no password": {
			Share{
				ID: "a1b2c3d4",
			},
			"",
			nil,
		},
		"password no auth": {
			Share{
				ID:       "a1b2c3d4",
				Password: string(password),
			},
			"",
			errors.New("empty authorization header"),
		},
		"invalid authorization": {
			Share{
				ID:       "a1b2c3d4",
				Password: string(password),
			},
			"invalid",
			errors.New("illegal base64 data at input byte 4"),
		},
		"invalid format": {
			Share{
				ID:       "a1b2c3d4",
				Password: string(password),
			},
			fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("test"))),
			errors.New("invalid format for basic auth"),
		},
		"invalid password": {
			Share{
				ID:       "a1b2c3d4",
				Password: string(password),
			},
			fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("test:password"))),
			errors.New("invalid credentials"),
		},
		"valid": {
			Share{
				ID:       "a1b2c3d4",
				Password: string(password),
			},
			fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("test:test"))),
			nil,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			err := tc.share.CheckPassword(tc.header)

			failed := false

			if err == nil && tc.want != nil {
				failed = true
			} else if err != nil && tc.want == nil {
				failed = true
			} else if err != nil && err.Error() != tc.want.Error() {
				failed = true
			}

			if failed {
				t.Errorf("CheckPassword() = `%s`, want `%s`", err, tc.want)
			}
		})
	}
}
