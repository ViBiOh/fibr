package fibr

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/ViBiOh/auth/v2/pkg/auth"
	"github.com/ViBiOh/auth/v2/pkg/ident"
	"github.com/ViBiOh/fibr/pkg/crud/crudtest"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func TestParseShare(t *testing.T) {
	type args struct {
		request             *provider.Request
		authorizationHeader string
	}

	var cases = []struct {
		intention string
		instance  app
		args      args
		want      *provider.Request
		wantErr   error
	}{
		{
			"no share",
			app{
				crudApp: crudtest.New(),
			},
			args{
				request: &provider.Request{
					Path:     "/",
					CanEdit:  false,
					CanShare: false,
					Display:  "grid",
				},
			},
			&provider.Request{
				Path:     "/",
				CanEdit:  false,
				CanShare: false,
				Display:  "grid",
			},
			nil,
		},
		{
			"passwordless share",
			app{
				crudApp: crudtest.New(),
			},
			args{
				request: &provider.Request{
					Path:     "/a1b2c3d4f5/index.html",
					CanEdit:  false,
					CanShare: false,
					Display:  "grid",
				},
			},
			&provider.Request{
				Path:     "/index.html",
				CanEdit:  false,
				CanShare: false,
				Display:  "grid",
				Share:    crudtest.PasswordLessShare,
			},
			nil,
		},
		{
			"empty password",
			app{
				crudApp: crudtest.New(),
			},
			args{
				request: &provider.Request{
					Path:     "/f5d4c3b2a1/index.html",
					CanEdit:  false,
					CanShare: false,
					Display:  "grid",
				},
			},
			&provider.Request{
				Path:     "/f5d4c3b2a1/index.html",
				CanEdit:  false,
				CanShare: false,
				Display:  "grid",
			},
			errors.New("empty authorization header"),
		},
		{
			"valid",
			app{
				crudApp: crudtest.New(),
			},
			args{
				request: &provider.Request{
					Path:     "/f5d4c3b2a1/index.html",
					CanEdit:  false,
					CanShare: false,
					Display:  "grid",
				},
				authorizationHeader: fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("admin:password"))),
			},
			&provider.Request{
				Path:     "/index.html",
				CanEdit:  true,
				CanShare: false,
				Display:  "grid",
				Share:    crudtest.PasswordShare,
			},
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			gotErr := tc.instance.parseShare(tc.args.request, tc.args.authorizationHeader)

			failed := false

			if tc.wantErr == nil && gotErr != nil {
				failed = true
			} else if tc.wantErr != nil && gotErr == nil {
				failed = true
			} else if tc.wantErr != nil && !strings.Contains(gotErr.Error(), tc.wantErr.Error()) {
				failed = true
			} else if !reflect.DeepEqual(tc.args.request, tc.want) {
				failed = true
			}

			if failed {
				t.Errorf("parseShare() = (%+v, `%s`), want (%+v, `%s`)", tc.args.request, gotErr, tc.want, tc.wantErr)
			}
		})
	}
}

func TestConvertAuthenticationError(t *testing.T) {
	type args struct {
		err error
	}

	var cases = []struct {
		intention string
		args      args
		want      *provider.Error
	}{
		{
			"forbidden",
			args{
				err: fmt.Errorf("forbidden access: %w", auth.ErrForbidden),
			},
			provider.NewError(http.StatusForbidden, errors.New("you're not authorized to speak to me")),
		},
		{
			"malformed",
			args{
				err: fmt.Errorf("invalid access: %w", ident.ErrMalformedAuth),
			},
			provider.NewError(http.StatusBadRequest, errors.New("malformed auth")),
		},
		{
			"unauthorized",
			args{
				err: fmt.Errorf("invalid: %w", ident.ErrInvalidCredentials),
			},
			provider.NewError(http.StatusUnauthorized, errors.New("try again")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := convertAuthenticationError(tc.args.err); got.Status != tc.want.Status {
				t.Errorf("convertAuthenticationError() = %d, want %d", got.Status, tc.want.Status)
			}
		})
	}
}
