package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrossOriginProtection(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		method     string
		headers    map[string]string
		publicURL  string
		wantStatus int
	}{
		"command line client": {
			http.MethodPost,
			nil,
			"https://fibr.example.com",
			http.StatusOK,
		},
		"same origin browser": {
			http.MethodPost,
			map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "https://fibr.example.com"},
			"https://fibr.example.com",
			http.StatusOK,
		},
		"cross site browser": {
			http.MethodPost,
			map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example.org"},
			"https://fibr.example.com",
			http.StatusForbidden,
		},
		"same site browser": {
			http.MethodDelete,
			map[string]string{"Sec-Fetch-Site": "same-site", "Origin": "https://evil.example.com"},
			"https://fibr.example.com",
			http.StatusForbidden,
		},
		"cross site browser without fetch metadata": {
			http.MethodPatch,
			map[string]string{"Origin": "https://evil.example.org"},
			"https://fibr.example.com",
			http.StatusForbidden,
		},
		"public url behind a proxy rewriting host": {
			http.MethodPost,
			map[string]string{"Origin": "https://fibr.example.com"},
			"https://fibr.example.com/prefix",
			http.StatusOK,
		},
		"cross site read": {
			http.MethodGet,
			map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example.org"},
			"https://fibr.example.com",
			http.StatusOK,
		},
		"no public url": {
			http.MethodPost,
			map[string]string{"Sec-Fetch-Site": "cross-site"},
			"",
			http.StatusForbidden,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			protection, err := newCrossOriginProtection(tc.publicURL)
			require.NoError(t, err)

			handler := protection.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(tc.method, "http://fibr.internal/", nil)
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tc.wantStatus, recorder.Code)
		})
	}
}
