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
	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/golang/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

var (
	invalidPath = "/invalid"
	adminPath   = "/admin"

	passwordLessShare = provider.Share{
		ID:       "a1b2c3d4f5",
		Edit:     false,
		RootName: "public",
		File:     false,
		Path:     "/public",
	}

	passwordHash, _ = bcrypt.GenerateFromPassword([]byte("password"), provider.BcryptCost)

	passwordShare = provider.Share{
		ID:       "f5d4c3b2a1",
		Edit:     true,
		RootName: "private",
		File:     false,
		Path:     "/private",
		Password: string(passwordHash),
	}
)

type authMiddlewareTest struct{}

func (amt authMiddlewareTest) Middleware(http.Handler) http.Handler {
	return nil
}

func (amt authMiddlewareTest) IsAuthenticated(r *http.Request, _ string) (ident.Provider, model.User, error) {
	if r.URL.Path == invalidPath {
		return nil, model.NoneUser, errors.New("authentication failed")
	}

	if r.URL.Path == adminPath {
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
			app{},
			args{
				request: &provider.Request{
					Path:     "/",
					CanEdit:  false,
					CanShare: false,
					Display:  provider.DefaultDisplay,
				},
			},
			&provider.Request{
				Path:     "/",
				CanEdit:  false,
				CanShare: false,
				Display:  provider.DefaultDisplay,
			},
			nil,
		},
		{
			"passwordless",
			app{},
			args{
				request: &provider.Request{
					Path:     "/a1b2c3d4f5/index.html",
					CanEdit:  false,
					CanShare: false,
					Display:  provider.DefaultDisplay,
				},
			},
			&provider.Request{
				Path:     "/index.html",
				CanEdit:  false,
				CanShare: false,
				Display:  provider.DefaultDisplay,
				Share:    passwordLessShare,
			},
			nil,
		},
		{
			"empty password",
			app{},
			args{
				request: &provider.Request{
					Path:     "/f5d4c3b2a1/index.html",
					CanEdit:  false,
					CanShare: false,
					Display:  provider.DefaultDisplay,
				},
			},
			&provider.Request{
				Path:     "/f5d4c3b2a1/index.html",
				CanEdit:  false,
				CanShare: false,
				Display:  provider.DefaultDisplay,
			},
			errors.New("empty authorization header"),
		},
		{
			"valid",
			app{},
			args{
				request: &provider.Request{
					Path:     "/f5d4c3b2a1/index.html",
					CanEdit:  false,
					CanShare: false,
					Display:  provider.DefaultDisplay,
				},
				authorizationHeader: fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("admin:password"))),
			},
			&provider.Request{
				Path:     "/index.html",
				CanEdit:  true,
				CanShare: false,
				Display:  provider.DefaultDisplay,
				Share:    passwordShare,
			},
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			metadataMock := mocks.NewMetadata(ctrl)
			tc.instance.metadataApp = metadataMock

			switch tc.intention {
			case "passwordless":
				metadataMock.EXPECT().GetShare(gomock.Any()).Return(passwordLessShare)
			case "empty password":
				fallthrough
			case "valid":
				metadataMock.EXPECT().GetShare(gomock.Any()).Return(passwordShare)
			default:
				metadataMock.EXPECT().GetShare(gomock.Any()).Return(provider.Share{})
			}

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
		want      error
	}{
		{
			"forbidden",
			args{
				err: fmt.Errorf("forbidden access: %w", auth.ErrForbidden),
			},
			httpModel.ErrForbidden,
		},
		{
			"malformed",
			args{
				err: fmt.Errorf("invalid access: %w", ident.ErrMalformedAuth),
			},
			httpModel.ErrInvalid,
		},
		{
			"unauthorized",
			args{
				err: fmt.Errorf("invalid: %w", ident.ErrInvalidCredentials),
			},
			httpModel.ErrUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := convertAuthenticationError(tc.args.err); !errors.Is(got, tc.want) {
				t.Errorf("convertAuthenticationError() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestParseRequest(t *testing.T) {
	adminRequestWithEmptyCookie := httptest.NewRequest(http.MethodGet, adminPath, nil)
	adminRequestWithEmptyCookie.AddCookie(&http.Cookie{
		Name:  "list_layout_paths",
		Value: "",
	})

	adminRequestWithCookie := httptest.NewRequest(http.MethodGet, adminPath, nil)
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
		wantErr   error
	}{
		{
			"error",
			app{},
			args{
				r: httptest.NewRequest(http.MethodGet, "/f5d4c3b2a1/", nil),
			},
			provider.Request{
				Path:     "/f5d4c3b2a1/",
				Display:  provider.DefaultDisplay,
				CanEdit:  false,
				CanShare: false,
			},
			httpModel.ErrUnauthorized,
		},
		{
			"share",
			app{},
			args{
				r: httptest.NewRequest(http.MethodGet, "/a1b2c3d4f5/", nil),
			},
			provider.Request{
				Path:     "/",
				Display:  provider.DefaultDisplay,
				CanEdit:  false,
				CanShare: false,
				Share:    passwordLessShare,
			},
			nil,
		},
		{
			"no auth",
			app{},
			args{
				r: httptest.NewRequest(http.MethodGet, "/", nil),
			},
			provider.Request{
				Path:     "/",
				Display:  provider.DefaultDisplay,
				CanEdit:  true,
				CanShare: false,
			},
			nil,
		},
		{
			"invalid auth",
			app{
				loginApp: authMiddlewareTest{},
			},
			args{
				r: httptest.NewRequest(http.MethodGet, invalidPath, nil),
			},
			provider.Request{
				Path:     invalidPath,
				Display:  provider.DefaultDisplay,
				CanEdit:  false,
				CanShare: false,
			},
			httpModel.ErrUnauthorized,
		},
		{
			"non admin user",
			app{
				loginApp: authMiddlewareTest{},
			},
			args{
				r: httptest.NewRequest(http.MethodGet, "/guest", nil),
			},
			provider.Request{
				Path:     "/guest",
				Display:  provider.DefaultDisplay,
				CanEdit:  false,
				CanShare: false,
			},
			nil,
		},
		{
			"admin user",
			app{
				loginApp: authMiddlewareTest{},
			},
			args{
				r: httptest.NewRequest(http.MethodGet, adminPath, nil),
			},
			provider.Request{
				Path:     adminPath,
				Display:  provider.DefaultDisplay,
				CanEdit:  true,
				CanShare: true,
			},
			nil,
		},
		{
			"empty cookie",
			app{
				loginApp: authMiddlewareTest{},
			},
			args{
				r: adminRequestWithEmptyCookie,
			},
			provider.Request{
				Path:     adminPath,
				Display:  provider.DefaultDisplay,
				CanEdit:  true,
				CanShare: false,
			},
			nil,
		},
		{
			"cookie value",
			app{
				loginApp: authMiddlewareTest{},
			},
			args{
				r: adminRequestWithCookie,
			},
			provider.Request{
				Path:     adminPath,
				Display:  provider.DefaultDisplay,
				CanEdit:  true,
				CanShare: false,
				Preferences: provider.Preferences{
					ListLayoutPath: []string{"assets", "documents/monthly"},
				},
			},
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			crudMock := mocks.NewCrud(ctrl)
			metadataMock := mocks.NewMetadata(ctrl)

			tc.instance.crudApp = crudMock
			tc.instance.metadataApp = metadataMock

			switch tc.intention {
			case "no auth":
				metadataMock.EXPECT().GetShare(gomock.Any()).Return(provider.Share{})
				metadataMock.EXPECT().Enabled().Return(false)
			case "admin user":
				metadataMock.EXPECT().Enabled().Return(true)
				metadataMock.EXPECT().GetShare(gomock.Any()).Return(provider.Share{})
			case "empty cookie", "cookie value":
				metadataMock.EXPECT().Enabled().Return(false)
				metadataMock.EXPECT().GetShare(gomock.Any()).Return(provider.Share{})
			case "invalid auth", "non admin user":
				metadataMock.EXPECT().GetShare(gomock.Any()).Return(provider.Share{})
			case "error":
				metadataMock.EXPECT().GetShare(gomock.Any()).Return(passwordShare)
			case "share":
				metadataMock.EXPECT().GetShare(gomock.Any()).Return(passwordLessShare)
			}

			got, gotErr := tc.instance.parseRequest(tc.args.r)

			failed := false

			if tc.wantErr != nil && gotErr == nil {
				failed = true
			} else if tc.wantErr == nil && gotErr != nil {
				failed = true
			} else if tc.wantErr != nil && !errors.Is(gotErr, tc.wantErr) {
				failed = true
			} else if !reflect.DeepEqual(got, tc.want) {
				failed = true
			}

			if failed {
				t.Errorf("parseRequest() = (%+v, `%s`), want (%+v, `%s`)", got, gotErr, tc.want, tc.wantErr)
			}
		})
	}
}
