package fibr

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/ViBiOh/auth/v2/pkg/auth"
	"github.com/ViBiOh/auth/v2/pkg/ident"
	"github.com/ViBiOh/auth/v2/pkg/model"
	"github.com/ViBiOh/fibr/pkg/crud/crudtest"
	"github.com/ViBiOh/fibr/pkg/provider"
)

type authMiddlewareTest struct{}

func (amt authMiddlewareTest) Middleware(http.Handler) http.Handler {
	return nil
}

func (amt authMiddlewareTest) IsAuthenticated(r *http.Request, _ string) (ident.Provider, model.User, error) {
	if r.URL.Path == "/invalid" {
		return nil, model.NoneUser, errors.New("authentication failed")
	}

	if r.URL.Path == "/admin" {
		return nil, model.User{ID: 8000}, nil
	}

	return nil, model.NoneUser, nil
}

func (amt authMiddlewareTest) HasProfile(_ context.Context, user model.User, _ string) bool {
	return user.ID == 8000
}

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

func TestParseRequest(t *testing.T) {
	adminRequestWithEmptyCookie := httptest.NewRequest(http.MethodGet, "/admin", nil)
	adminRequestWithEmptyCookie.AddCookie(&http.Cookie{
		Name:  "list_layout_paths",
		Value: "",
	})

	adminRequestWithCookie := httptest.NewRequest(http.MethodGet, "/admin", nil)
	adminRequestWithCookie.AddCookie(&http.Cookie{
		Name:  "list_layout_paths",
		Value: "assets,documents/monthly",
	})

	type args struct {
		r *http.Request
	}

	var cases = []struct {
		intention string
		instance  app
		args      args
		want      provider.Request
		wantErr   *provider.Error
	}{
		{
			"share error",
			app{
				crudApp: crudtest.New(),
			},
			args{
				r: httptest.NewRequest(http.MethodGet, "/f5d4c3b2a1/", nil),
			},
			provider.Request{
				Path:     "/f5d4c3b2a1/",
				CanEdit:  false,
				CanShare: false,
			},
			provider.NewError(http.StatusUnauthorized, errors.New("test")),
		},
		{
			"share",
			app{
				crudApp: crudtest.New(),
			},
			args{
				r: httptest.NewRequest(http.MethodGet, "/a1b2c3d4f5/", nil),
			},
			provider.Request{
				Path:     "/",
				CanEdit:  false,
				CanShare: false,
				Share:    crudtest.PasswordLessShare,
			},
			nil,
		},
		{
			"no auth",
			app{
				crudApp: crudtest.New(),
			},
			args{
				r: httptest.NewRequest(http.MethodGet, "/", nil),
			},
			provider.Request{
				Path:     "/",
				CanEdit:  true,
				CanShare: true,
			},
			nil,
		},
		{
			"invalid auth",
			app{
				crudApp:  crudtest.New(),
				loginApp: authMiddlewareTest{},
			},
			args{
				r: httptest.NewRequest(http.MethodGet, "/invalid", nil),
			},
			provider.Request{
				Path:     "/invalid",
				CanEdit:  false,
				CanShare: false,
			},
			provider.NewError(http.StatusUnauthorized, errors.New("test")),
		},
		{
			"non admin user",
			app{
				crudApp:  crudtest.New(),
				loginApp: authMiddlewareTest{},
			},
			args{
				r: httptest.NewRequest(http.MethodGet, "/guest", nil),
			},
			provider.Request{
				Path:     "/guest",
				CanEdit:  false,
				CanShare: false,
			},
			nil,
		},
		{
			"admin user",
			app{
				crudApp:  crudtest.New(),
				loginApp: authMiddlewareTest{},
			},
			args{
				r: httptest.NewRequest(http.MethodGet, "/admin", nil),
			},
			provider.Request{
				Path:     "/admin",
				CanEdit:  true,
				CanShare: true,
			},
			nil,
		},
		{
			"empty cookie",
			app{
				crudApp:  crudtest.New(),
				loginApp: authMiddlewareTest{},
			},
			args{
				r: adminRequestWithEmptyCookie,
			},
			provider.Request{
				Path:     "/admin",
				CanEdit:  true,
				CanShare: true,
			},
			nil,
		},
		{
			"cookie value",
			app{
				crudApp:  crudtest.New(),
				loginApp: authMiddlewareTest{},
			},
			args{
				r: adminRequestWithCookie,
			},
			provider.Request{
				Path:     "/admin",
				CanEdit:  true,
				CanShare: true,
				Preferences: provider.Preferences{
					ListLayoutPath: []string{"assets", "documents/monthly"},
				},
			},
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			got, gotErr := tc.instance.parseRequest(tc.args.r)

			failed := false

			if tc.wantErr != nil && gotErr == nil {
				failed = true
			} else if tc.wantErr == nil && gotErr != nil {
				failed = true
			} else if tc.wantErr != nil && tc.wantErr.Status != gotErr.Status {
				failed = true
			} else if !reflect.DeepEqual(got, tc.want) {
				failed = true
			}

			if failed {
				t.Errorf("parseRequest() = (%+v, %+v), want (%+v, %+v)", got, gotErr, tc.want, tc.wantErr)
			}
		})
	}
}
