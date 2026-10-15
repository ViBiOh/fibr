package push

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDelete(t *testing.T) {
	t.Parallel()

	first := newTestSubscription(t, "https://fcm.googleapis.com/fcm/send/first")
	second := newTestSubscription(t, "https://fcm.googleapis.com/fcm/send/second")
	third := newTestSubscription(t, "https://updates.push.services.mozilla.com/wpush/v2/third")

	cases := map[string]struct {
		endpoint string
		want     []string
	}{
		"first": {
			first.Endpoint,
			[]string{second.Endpoint, third.Endpoint},
		},
		"middle": {
			second.Endpoint,
			[]string{first.Endpoint, third.Endpoint},
		},
		"last": {
			third.Endpoint,
			[]string{first.Endpoint, second.Endpoint},
		},
		"unknown": {
			"https://fcm.googleapis.com/fcm/send/unknown",
			[]string{first.Endpoint, second.Endpoint, third.Endpoint},
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			instance := newTestService(t, DefaultAllowedHosts, 0)

			for _, subscription := range []Subscription{first, second, third} {
				require.NoError(t, instance.Add(t.Context(), subscription))
			}

			require.NoError(t, instance.Delete(t.Context(), tc.endpoint))

			stored, err := instance.get(t.Context())
			require.NoError(t, err)

			endpoints := make([]string, len(stored))
			for index, subscription := range stored {
				endpoints[index] = subscription.Endpoint
			}

			assert.Equal(t, tc.want, endpoints)
		})
	}
}

func TestDeleteWithoutStorageFile(t *testing.T) {
	t.Parallel()

	instance := newTestService(t, DefaultAllowedHosts, 0)

	assert.NoError(t, instance.Delete(t.Context(), chromeEndpoint))
}

func TestFind(t *testing.T) {
	t.Parallel()

	instance := newTestService(t, DefaultAllowedHosts, 0)
	subscription := newTestSubscription(t, chromeEndpoint)

	require.NoError(t, instance.Add(t.Context(), subscription))

	found, err := instance.Find(t.Context(), chromeEndpoint)
	require.NoError(t, err)
	assert.Equal(t, subscription.Endpoint, found.Endpoint)

	_, err = instance.Find(t.Context(), "https://fcm.googleapis.com/fcm/send/unknown")
	assert.ErrorIs(t, err, ErrNotFound)
}
