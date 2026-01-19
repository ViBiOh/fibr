package fibr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/ViBiOh/auth/v3/pkg/argon"
	authModel "github.com/ViBiOh/auth/v3/pkg/model"
	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"go.uber.org/mock/gomock"
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

	passwordHash, _ = argon.GenerateFromPassword("password")

	passwordShare = provider.Share{
		ID:       "f5d4c3b2a1",
		Edit:     true,
		RootName: "private",
		File:     false,
		Path:     "/private",
		Password: string(passwordHash),
	}
)

func TestParseShare(t *testing.T) {
	type args struct {
		request  *provider.Request
		password string
	}

	cases := map[string]struct {
		instance Service
		args     args
		want     *provider.Request
		wantErr  error
	}{
		"no share": {
			Service{},
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
		"passwordless": {
			Service{},
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
		"empty password": {
			Service{},
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
			errors.New("empty password authorization"),
		},
		"valid": {
			Service{},
			args{
				request: &provider.Request{
					Path:     "/f5d4c3b2a1/index.html",
					CanEdit:  false,
					CanShare: false,
					Display:  provider.DefaultDisplay,
				},
				password: "password",
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

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			shareMock := mocks.NewShareManager(ctrl)
			tc.instance.share = shareMock

			switch intention {
			case "passwordless":
				shareMock.EXPECT().Get(gomock.Any()).Return(passwordLessShare)
			case "empty password":
				fallthrough
			case "valid":
				shareMock.EXPECT().Get(gomock.Any()).Return(passwordShare)
			default:
				shareMock.EXPECT().Get(gomock.Any()).Return(provider.Share{})
			}

			gotErr := tc.instance.parseShare(context.Background(), tc.args.request, tc.args.password)

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

	cases := map[string]struct {
		args args
		want error
	}{
		"forbidden": {
			args{
				err: fmt.Errorf("no secret defense: %w", authModel.ErrForbidden),
			},
			httpModel.ErrForbidden,
		},
		"malformed": {
			args{
				err: fmt.Errorf("invalid access: %w", authModel.ErrMalformedContent),
			},
			httpModel.ErrUnauthorized,
		},
		"unauthorized": {
			args{
				err: fmt.Errorf("invalid: %w", authModel.ErrInvalidCredentials),
			},
			httpModel.ErrUnauthorized,
		},
		"other": {
			args{
				err: fmt.Errorf("invalid: %w", authModel.ErrUnavailableService),
			},
			httpModel.ErrInvalid,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			if got := convertAuthenticationError(tc.args.err); !errors.Is(got, tc.want) {
				t.Errorf("convertAuthenticationError() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestParseRequest(t *testing.T) {
	adminRequestWithEmptyCookie := httptest.NewRequest(http.MethodGet, adminPath, nil)
	adminRequestWithEmptyCookie.AddCookie(&http.Cookie{
		Name:  provider.LayoutPathsCookieName,
		Value: "",
	})

	adminRequestWithCookie := httptest.NewRequest(http.MethodGet, adminPath, nil)
	adminRequestWithCookie.AddCookie(&http.Cookie{
		Name:  provider.LayoutPathsCookieName,
		Value: "assets|list,documents/monthly|story",
	})

	guestRequest := httptest.NewRequest(http.MethodGet, "/guest", nil)
	guestRequest.SetBasicAuth("guest", "guest")

	adminRequest := httptest.NewRequest(http.MethodGet, adminPath, nil)
	adminRequest.SetBasicAuth("admin", "admin")

	type args struct {
		r *http.Request
	}

	cases := map[string]struct {
		instance Service
		args     args
		want     provider.Request
		wantErr  error
	}{
		"error": {
			Service{},
			args{
				r: httptest.NewRequest(http.MethodGet, "/f5d4c3b2a1/", nil),
			},
			provider.Request{
				Path:     "/f5d4c3b2a1/",
				CanEdit:  false,
				CanShare: false,
			},
			httpModel.ErrUnauthorized,
		},
		"share": {
			Service{},
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
		"no auth": {
			Service{},
			args{
				r: httptest.NewRequest(http.MethodGet, "/", nil),
			},
			provider.Request{
				Path:       "/",
				Display:    provider.DefaultDisplay,
				CanEdit:    true,
				CanShare:   true,
				CanWebhook: true,
			},
			nil,
		},
		"invalid auth": {
			Service{},
			args{
				r: httptest.NewRequest(http.MethodGet, invalidPath, nil),
			},
			provider.Request{
				Path:     "/",
				Item:     "invalid",
				Display:  provider.DefaultDisplay,
				CanEdit:  false,
				CanShare: false,
			},
			httpModel.ErrUnauthorized,
		},
		"non admin user": {
			Service{},
			args{
				r: guestRequest,
			},
			provider.Request{
				Path:       "/",
				Item:       "guest",
				Display:    provider.DefaultDisplay,
				CanEdit:    false,
				CanShare:   false,
				CanWebhook: false,
			},
			nil,
		},
		"admin user": {
			Service{},
			args{
				r: adminRequest,
			},
			provider.Request{
				Path:       "/",
				Item:       "admin",
				Display:    provider.DefaultDisplay,
				CanEdit:    true,
				CanShare:   true,
				CanWebhook: true,
			},
			nil,
		},
		"empty cookie": {
			Service{},
			args{
				r: adminRequestWithEmptyCookie,
			},
			provider.Request{
				Path:       "/",
				Item:       "admin",
				Display:    provider.DefaultDisplay,
				CanEdit:    true,
				CanShare:   true,
				CanWebhook: true,
			},
			nil,
		},
		"cookie value": {
			Service{},
			args{
				r: adminRequestWithCookie,
			},
			provider.Request{
				Path:       "/",
				Item:       "admin",
				Display:    provider.DefaultDisplay,
				CanEdit:    true,
				CanShare:   true,
				CanWebhook: true,
				Preferences: provider.Preferences{
					LayoutPaths: map[string]provider.Display{
						"assets":            provider.ListDisplay,
						"documents/monthly": provider.StoryDisplay,
					},
				},
			},
			nil,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			crudMock := mocks.NewCrud(ctrl)
			shareMock := mocks.NewShareManager(ctrl)
			webhookMock := mocks.NewWebhookManager(ctrl)
			loginMock := mocks.NewAuth(ctrl)

			tc.instance.crud = crudMock
			tc.instance.share = shareMock
			tc.instance.webhook = webhookMock

			switch intention {
			case "no auth":
				shareMock.EXPECT().Get(gomock.Any()).Return(provider.Share{})

			case "admin user":
				shareMock.EXPECT().Get(gomock.Any()).Return(provider.Share{})

			case "empty cookie", "cookie value":
				shareMock.EXPECT().Get(gomock.Any()).Return(provider.Share{})

			case "invalid auth", "non admin user":
				shareMock.EXPECT().Get(gomock.Any()).Return(provider.Share{})

			case "error":
				shareMock.EXPECT().Get(gomock.Any()).Return(passwordShare)

			case "share":
				shareMock.EXPECT().Get(gomock.Any()).Return(passwordLessShare)
			}

			switch intention {
			case "invalid auth":
				tc.instance.login = loginMock

			case "non admin user":
				tc.instance.login = loginMock
				loginMock.EXPECT().GetBasicUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(authModel.User{}, nil)
				loginMock.EXPECT().IsAuthorized(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)

			case "admin user":
				tc.instance.login = loginMock
				loginMock.EXPECT().GetBasicUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(authModel.User{}, nil)
				loginMock.EXPECT().IsAuthorized(gomock.Any(), gomock.Any(), gomock.Any()).Return(true)
			}

			got, gotErr := tc.instance.parseRequest(httptest.NewRecorder(), tc.args.r)

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
				t.Errorf("parseRequest() = (%#v, `%s`), want (%#v, `%s`)", got, gotErr, tc.want, tc.wantErr)
			}
		})
	}
}
