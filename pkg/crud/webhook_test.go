package crud

import (
	"encoding/base64"
	"flag"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/push"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	pushEndpoint      = "https://fcm.googleapis.com/fcm/send/abcdef:123"
	otherPushEndpoint = "https://updates.push.services.mozilla.com/wpush/v2/other"
)

func newPushTestService(t *testing.T) (*Service, *push.Service) {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".fibr"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "photos"), 0o755))

	storage, err := filesystem.New(root)
	require.NoError(t, err)

	flagSet := flag.NewFlagSet("push", flag.ContinueOnError)
	config := push.Flags(flagSet, "")
	require.NoError(t, flagSet.Parse([]string{"-publicKey=public", "-privateKey=" + base64.RawURLEncoding.EncodeToString(make([]byte, 32))}))

	pushService, err := push.New(config, storage, exclusive.New(nil))
	require.NoError(t, err)

	require.NoError(t, pushService.Add(t.Context(), push.Subscription{Endpoint: pushEndpoint, PublicKey: "key", Auth: "auth"}))

	return &Service{storage: storage, rawStorage: storage, pushService: pushService}, pushService
}

func newWebhookRequest(form url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/photos/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/plain")

	return req
}

func TestCreatePushWebhook(t *testing.T) {
	t.Parallel()

	pushForm := func(endpoint string) url.Values {
		return url.Values{"kind": {"push"}, "url": {endpoint}, "types": {"upload"}, "recursive": {"true"}}
	}

	cases := map[string]struct {
		request     provider.Request
		form        url.Values
		withoutPush bool
		wantCreate  bool
		wantStatus  int
	}{
		"registered subscription": {
			provider.Request{Path: "/photos/"},
			pushForm(pushEndpoint),
			false,
			true,
			http.StatusCreated,
		},
		"subscription that was never registered": {
			provider.Request{Path: "/photos/"},
			pushForm(otherPushEndpoint),
			false,
			false,
			http.StatusBadRequest,
		},
		"arbitrary internal url": {
			provider.Request{Path: "/photos/"},
			pushForm("https://169.254.169.254/latest/meta-data"),
			false,
			false,
			http.StatusBadRequest,
		},
		"push notifications not configured": {
			provider.Request{Path: "/photos/"},
			pushForm(pushEndpoint),
			true,
			false,
			http.StatusBadRequest,
		},
		"raw webhook without right": {
			provider.Request{Path: "/photos/"},
			url.Values{"kind": {"raw"}, "url": {"https://example.com/hook"}, "types": {"upload"}},
			false,
			false,
			http.StatusForbidden,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			instance, _ := newPushTestService(t)
			webhookMock := mocks.NewWebhookManager(gomock.NewController(t))
			instance.webhook = webhookMock

			if tc.withoutPush {
				instance.pushService = nil
			}

			if tc.wantCreate {
				webhookMock.EXPECT().Create(gomock.Any(), "/photos/", true, provider.Push, pushEndpoint, []provider.EventType{provider.UploadEvent}).Return("webhook-id", nil)
			}

			recorder := httptest.NewRecorder()

			instance.createWebhook(recorder, newWebhookRequest(tc.form), tc.request)

			assert.Equal(t, tc.wantStatus, recorder.Code)

			if tc.wantCreate {
				assert.Equal(t, "webhook-id", recorder.Body.String())
			}
		})
	}
}

func TestDeletePushWebhook(t *testing.T) {
	t.Parallel()

	pushWebhook := provider.Webhook{ID: "webhook-id", Pathname: "/photos/", URL: pushEndpoint, Kind: provider.Push}
	rawWebhook := provider.Webhook{ID: "webhook-id", Pathname: "/photos/", URL: "https://example.com/hook", Kind: provider.Raw}
	otherDirectoryWebhook := provider.Webhook{ID: "webhook-id", Pathname: "/videos/", URL: pushEndpoint, Kind: provider.Push}

	cases := map[string]struct {
		request          provider.Request
		webhook          provider.Webhook
		url              string
		remainingForURL  []provider.Webhook
		wantDelete       bool
		wantStatus       int
		wantSubscription bool
	}{
		"owner with last webhook of the subscription": {
			provider.Request{Path: "/photos/"},
			pushWebhook,
			pushEndpoint,
			nil,
			true,
			http.StatusNoContent,
			false,
		},
		"owner with other webhooks on the subscription": {
			provider.Request{Path: "/photos/"},
			pushWebhook,
			pushEndpoint,
			[]provider.Webhook{otherDirectoryWebhook},
			true,
			http.StatusNoContent,
			true,
		},
		"without the subscription endpoint": {
			provider.Request{Path: "/photos/"},
			pushWebhook,
			"",
			nil,
			false,
			http.StatusForbidden,
			true,
		},
		"with the wrong subscription endpoint": {
			provider.Request{Path: "/photos/"},
			pushWebhook,
			otherPushEndpoint,
			nil,
			false,
			http.StatusForbidden,
			true,
		},
		"webhook of another directory": {
			provider.Request{Path: "/photos/"},
			otherDirectoryWebhook,
			pushEndpoint,
			nil,
			false,
			http.StatusForbidden,
			true,
		},
		"not a push webhook": {
			provider.Request{Path: "/photos/"},
			rawWebhook,
			rawWebhook.URL,
			nil,
			false,
			http.StatusForbidden,
			true,
		},
		"unknown webhook": {
			provider.Request{Path: "/photos/"},
			provider.Webhook{},
			pushEndpoint,
			nil,
			false,
			http.StatusForbidden,
			true,
		},
		"administrator deletes any webhook": {
			provider.Request{Path: "/photos/", CanWebhook: true},
			rawWebhook,
			"",
			nil,
			true,
			http.StatusNoContent,
			true,
		},
		"administrator deletes push webhook of another directory": {
			provider.Request{Path: "/photos/", CanWebhook: true},
			otherDirectoryWebhook,
			"",
			nil,
			true,
			http.StatusNoContent,
			false,
		},
		"administrator on unknown webhook": {
			provider.Request{Path: "/photos/", CanWebhook: true},
			provider.Webhook{},
			"",
			nil,
			false,
			http.StatusNotFound,
			true,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			instance, pushService := newPushTestService(t)
			webhookMock := mocks.NewWebhookManager(gomock.NewController(t))
			instance.webhook = webhookMock

			webhookMock.EXPECT().Get("webhook-id").Return(tc.webhook)

			if tc.wantDelete {
				webhookMock.EXPECT().Delete(gomock.Any(), "webhook-id").Return(nil)

				if tc.webhook.Kind == provider.Push {
					webhookMock.EXPECT().FindByURL(tc.webhook.URL).Return(tc.remainingForURL)
				}
			}

			form := url.Values{"id": {"webhook-id"}, "type": {"webhook"}, "method": {"DELETE"}}
			if len(tc.url) != 0 {
				form.Set("url", tc.url)
			}

			recorder := httptest.NewRecorder()

			instance.deleteWebhook(recorder, newWebhookRequest(form), tc.request)

			assert.Equal(t, tc.wantStatus, recorder.Code)

			_, err := pushService.Find(t.Context(), pushEndpoint)
			if tc.wantSubscription {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, push.ErrNotFound)
			}
		})
	}
}

func TestPostPushWithoutPushService(t *testing.T) {
	t.Parallel()

	instance := &Service{}

	req := httptest.NewRequest(http.MethodPost, "/photos/?push", strings.NewReader("{}"))
	req.Header.Set("Accept", "text/plain")

	recorder := httptest.NewRecorder()

	instance.Post(recorder, req, provider.Request{Path: "/photos/"})

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}
