package webhook

import (
	"testing"

	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/registry"
)

func BenchmarkList(b *testing.B) {
	instance := &Service{
		webhooks: registry.New[provider.Webhook](nil, exclusive.New(nil), nil, webhookFilename, "channel"),
	}

	instance.webhooks.Patch(func(webhooks map[string]provider.Webhook) {
		for _, id := range []string{"abcdef123456", "a1b2c3d4e5f6", "123456abcdef", "6f5e4d3c2b1a"} {
			webhooks[id] = provider.Webhook{
				ID:       id,
				URL:      "http://127.0.0.1",
				Pathname: "/website",
				Types:    []provider.EventType{provider.AccessEvent},
			}
		}
	})

	for b.Loop() {
		instance.List()
	}
}
