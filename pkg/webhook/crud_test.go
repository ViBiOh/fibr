package webhook

import (
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func BenchmarkList(b *testing.B) {
	instance := &App{
		webhooks: map[string]provider.Webhook{
			"abcdef123456": {
				ID:       "abcdef123456",
				URL:      "http://localhost",
				Pathname: "/website",
				Types:    []provider.EventType{provider.AccessEvent},
			},
			"a1b2c3d4e5f6": {
				ID:       "a1b2c3d4e5f6",
				URL:      "http://localhost",
				Pathname: "/website",
				Types:    []provider.EventType{provider.AccessEvent},
			},
			"123456abcdef": {
				ID:       "123456abcdef",
				URL:      "http://localhost",
				Pathname: "/website",
				Types:    []provider.EventType{provider.AccessEvent},
			},
			"6f5e4d3c2b1a": {
				ID:       "6f5e4d3c2b1a",
				URL:      "http://localhost",
				Pathname: "/website",
				Types:    []provider.EventType{provider.AccessEvent},
			},
		},
	}

	for i := 0; i < b.N; i++ {
		instance.List()
	}
}
