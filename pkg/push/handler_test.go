package push

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandle(t *testing.T) {
	t.Parallel()

	valid := newTestSubscription(t, chromeEndpoint)

	oversized := valid
	oversized.Endpoint = "https://fcm.googleapis.com/" + strings.Repeat("a", maxSubscriptionBodySize)

	cases := map[string]struct {
		body             string
		maxSubscriptions int
		seeded           int
		wantStatus       int
		wantStored       int
	}{
		"valid": {
			mustMarshal(t, valid),
			10,
			0,
			http.StatusOK,
			1,
		},
		"endpoint targeting an internal service": {
			mustMarshal(t, Subscription{Endpoint: "https://169.254.169.254/latest", PublicKey: valid.PublicKey, Auth: valid.Auth}),
			10,
			0,
			http.StatusBadRequest,
			0,
		},
		"endpoint over http": {
			mustMarshal(t, Subscription{Endpoint: "http://fcm.googleapis.com/fcm/send/abc", PublicKey: valid.PublicKey, Auth: valid.Auth}),
			10,
			0,
			http.StatusBadRequest,
			0,
		},
		"invalid keys": {
			mustMarshal(t, Subscription{Endpoint: chromeEndpoint, PublicKey: "AAAA", Auth: "AAAA"}),
			10,
			0,
			http.StatusBadRequest,
			0,
		},
		"not json": {
			"<html>",
			10,
			0,
			http.StatusBadRequest,
			0,
		},
		"body too large": {
			mustMarshal(t, oversized),
			10,
			0,
			http.StatusBadRequest,
			0,
		},
		"too many subscriptions": {
			mustMarshal(t, valid),
			1,
			1,
			http.StatusTooManyRequests,
			1,
		},
		"no limit": {
			mustMarshal(t, valid),
			0,
			1,
			http.StatusOK,
			2,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			instance := newTestService(t, DefaultAllowedHosts, tc.maxSubscriptions)

			for range tc.seeded {
				require.NoError(t, instance.Add(t.Context(), newTestSubscription(t, chromeEndpoint)))
			}

			req := httptest.NewRequest(http.MethodPost, "/?push", strings.NewReader(tc.body))
			recorder := httptest.NewRecorder()

			instance.Handle(recorder, req)

			assert.Equal(t, tc.wantStatus, recorder.Code)

			stored, err := instance.get(t.Context())
			require.NoError(t, err)
			assert.Len(t, stored, tc.wantStored)
		})
	}
}

func TestHandleSameSubscriptionTwice(t *testing.T) {
	t.Parallel()

	instance := newTestService(t, DefaultAllowedHosts, 10)
	body := mustMarshal(t, newTestSubscription(t, chromeEndpoint))

	for range 3 {
		recorder := httptest.NewRecorder()
		instance.Handle(recorder, httptest.NewRequest(http.MethodPost, "/?push", strings.NewReader(body)))

		assert.Equal(t, http.StatusOK, recorder.Code)
	}

	stored, err := instance.get(t.Context())
	require.NoError(t, err)
	assert.Len(t, stored, 1)
}

func mustMarshal(t *testing.T, subscription Subscription) string {
	t.Helper()

	payload, err := json.Marshal(subscription)
	require.NoError(t, err)

	return string(payload)
}
