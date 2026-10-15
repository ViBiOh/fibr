package push

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const chromeEndpoint = "https://fcm.googleapis.com/fcm/send/abcdef:123"

func newTestService(t *testing.T, allowedHosts []string, maxSubscriptions int) *Service {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".fibr"), 0o755))

	storage, err := filesystem.New(root)
	require.NoError(t, err)

	return &Service{
		storage:          storage,
		exclusive:        exclusive.New(nil),
		allowedHosts:     allowedHosts,
		maxSubscriptions: maxSubscriptions,
	}
}

func newTestSubscription(t *testing.T, endpoint string) Subscription {
	t.Helper()

	key, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)

	auth := make([]byte, authSecretSize)
	_, err = rand.Read(auth)
	require.NoError(t, err)

	return Subscription{
		Endpoint:  endpoint,
		PublicKey: base64.StdEncoding.EncodeToString(key.PublicKey().Bytes()),
		Auth:      base64.StdEncoding.EncodeToString(auth),
	}
}

func TestCheckEndpoint(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		endpoint     string
		allowedHosts []string
		wantErr      bool
	}{
		"chrome": {
			chromeEndpoint,
			DefaultAllowedHosts,
			false,
		},
		"firefox": {
			"https://updates.push.services.mozilla.com/wpush/v2/gAAAAAB",
			DefaultAllowedHosts,
			false,
		},
		"safari": {
			"https://web.push.apple.com/QGxyz",
			DefaultAllowedHosts,
			false,
		},
		"edge": {
			"https://par02p.notify.windows.com/w/?token=abc",
			DefaultAllowedHosts,
			false,
		},
		"explicit default port": {
			"https://fcm.googleapis.com:443/fcm/send/abc",
			DefaultAllowedHosts,
			false,
		},
		"host is case insensitive": {
			"https://FCM.GoogleAPIs.com/fcm/send/abc",
			DefaultAllowedHosts,
			false,
		},
		"trailing dot": {
			"https://fcm.googleapis.com./fcm/send/abc",
			DefaultAllowedHosts,
			false,
		},
		"empty": {
			"",
			DefaultAllowedHosts,
			true,
		},
		"too long": {
			"https://fcm.googleapis.com/" + strings.Repeat("a", maxEndpointSize),
			DefaultAllowedHosts,
			true,
		},
		"http scheme": {
			"http://fcm.googleapis.com/fcm/send/abc",
			DefaultAllowedHosts,
			true,
		},
		"no scheme": {
			"fcm.googleapis.com/fcm/send/abc",
			DefaultAllowedHosts,
			true,
		},
		"other scheme": {
			"file:///etc/passwd",
			DefaultAllowedHosts,
			true,
		},
		"credentials": {
			"https://fcm.googleapis.com@evil.example.com/fcm/send/abc",
			DefaultAllowedHosts,
			true,
		},
		"credentials with allowed host": {
			"https://user:pass@fcm.googleapis.com/fcm/send/abc",
			DefaultAllowedHosts,
			true,
		},
		"fragment trick": {
			"https://evil.example.com#@fcm.googleapis.com/",
			DefaultAllowedHosts,
			true,
		},
		"allowed host as subdomain of another": {
			"https://fcm.googleapis.com.evil.example.com/fcm/send/abc",
			DefaultAllowedHosts,
			true,
		},
		"exact host with extra subdomain": {
			"https://evil.fcm.googleapis.com/fcm/send/abc",
			DefaultAllowedHosts,
			true,
		},
		"suffix without dot": {
			"https://evilpush.apple.com/abc",
			DefaultAllowedHosts,
			true,
		},
		"wildcard needs a subdomain": {
			"https://push.apple.com/abc",
			DefaultAllowedHosts,
			true,
		},
		"custom port": {
			"https://fcm.googleapis.com:8443/fcm/send/abc",
			DefaultAllowedHosts,
			true,
		},
		"loopback": {
			"https://localhost/abc",
			DefaultAllowedHosts,
			true,
		},
		"loopback address": {
			"https://127.0.0.1/abc",
			DefaultAllowedHosts,
			true,
		},
		"metadata address": {
			"https://169.254.169.254/latest/meta-data",
			DefaultAllowedHosts,
			true,
		},
		"private service name": {
			"https://redis/abc",
			DefaultAllowedHosts,
			true,
		},
		"no allowed host": {
			chromeEndpoint,
			nil,
			true,
		},
		"custom exact host": {
			"https://push.example.com/abc",
			[]string{"push.example.com"},
			false,
		},
		"custom host trims spaces": {
			"https://push.example.com/abc",
			[]string{" push.example.com "},
			false,
		},
		"custom host does not allow defaults": {
			chromeEndpoint,
			[]string{"push.example.com"},
			true,
		},
		"any host": {
			"https://anything.example.com/abc",
			[]string{"*"},
			false,
		},
		"any host still requires https": {
			"http://anything.example.com/abc",
			[]string{"*"},
			true,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			err := (&Service{allowedHosts: tc.allowedHosts}).checkEndpoint(tc.endpoint)

			if tc.wantErr {
				assert.ErrorIs(t, err, ErrInvalidEndpoint)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	valid := newTestSubscription(t, chromeEndpoint)

	cases := map[string]struct {
		subscription Subscription
		wantErr      error
	}{
		"valid": {
			valid,
			nil,
		},
		"url safe encoding without padding": {
			Subscription{
				Endpoint:  valid.Endpoint,
				PublicKey: strings.TrimRight(strings.NewReplacer("+", "-", "/", "_").Replace(valid.PublicKey), "="),
				Auth:      strings.TrimRight(strings.NewReplacer("+", "-", "/", "_").Replace(valid.Auth), "="),
			},
			nil,
		},
		"endpoint not allowed": {
			Subscription{Endpoint: "https://127.0.0.1/", PublicKey: valid.PublicKey, Auth: valid.Auth},
			ErrInvalidEndpoint,
		},
		"missing auth": {
			Subscription{Endpoint: valid.Endpoint, PublicKey: valid.PublicKey},
			ErrInvalidSubscription,
		},
		"auth with wrong size": {
			Subscription{Endpoint: valid.Endpoint, PublicKey: valid.PublicKey, Auth: base64.StdEncoding.EncodeToString([]byte("short"))},
			ErrInvalidSubscription,
		},
		"auth not base64": {
			Subscription{Endpoint: valid.Endpoint, PublicKey: valid.PublicKey, Auth: "!!!"},
			ErrInvalidSubscription,
		},
		"missing public key": {
			Subscription{Endpoint: valid.Endpoint, Auth: valid.Auth},
			ErrInvalidSubscription,
		},
		"public key not a curve point": {
			Subscription{Endpoint: valid.Endpoint, PublicKey: base64.StdEncoding.EncodeToString(make([]byte, 65)), Auth: valid.Auth},
			ErrInvalidSubscription,
		},
		"public key not base64": {
			Subscription{Endpoint: valid.Endpoint, PublicKey: "!!!", Auth: valid.Auth},
			ErrInvalidSubscription,
		},
	}

	instance := &Service{allowedHosts: DefaultAllowedHosts}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			err := instance.validate(tc.subscription)

			if tc.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

func TestNotifyRejectsForbiddenEndpoint(t *testing.T) {
	t.Parallel()

	instance := &Service{allowedHosts: DefaultAllowedHosts}

	status, err := instance.Notify(t.Context(), newTestSubscription(t, "https://169.254.169.254/latest/meta-data"), Notification{Title: "title"})

	assert.ErrorIs(t, err, ErrInvalidEndpoint)
	assert.Zero(t, status)
}
