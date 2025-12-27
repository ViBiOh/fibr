package webhook

import (
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func BenchmarkList(b *testing.B) {
	instance := &Service{
		webhooks: map[string]provider.Webhook{
			"abcdef123456": {
				ID:       "abcdef123456",
				URL:      "http://127.0.0.1",
				Pathname: "/website",
				Types:    []provider.EventType{provider.AccessEvent},
			},
			"a1b2c3d4e5f6": {
				ID:       "a1b2c3d4e5f6",
				URL:      "http://127.0.0.1",
				Pathname: "/website",
				Types:    []provider.EventType{provider.AccessEvent},
			},
			"123456abcdef": {
				ID:       "123456abcdef",
				URL:      "http://127.0.0.1",
				Pathname: "/website",
				Types:    []provider.EventType{provider.AccessEvent},
			},
			"6f5e4d3c2b1a": {
				ID:       "6f5e4d3c2b1a",
				URL:      "http://127.0.0.1",
				Pathname: "/website",
				Types:    []provider.EventType{provider.AccessEvent},
			},
		},
	}

	for i := 0; i < b.N; i++ {
		instance.List()
	}
}
