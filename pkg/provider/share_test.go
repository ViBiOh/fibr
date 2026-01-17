package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/ViBiOh/auth/v3/pkg/argon"
)

func TestCheckPassword(t *testing.T) {
	password, err := argon.GenerateFromPassword("test")
	if err != nil {
		t.Errorf("create argon password: %s", err)
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
			errors.New("empty password authorization"),
		},
		"invalid password": {
			Share{
				ID:       "a1b2c3d4",
				Password: string(password),
			},
			"password",
			errors.New("invalid credentials"),
		},
		"valid": {
			Share{
				ID:       "a1b2c3d4",
				Password: string(password),
			},
			"test",
			nil,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			err := tc.share.CheckPassword(context.Background(), tc.header, nil)

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
