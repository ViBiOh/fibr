package crud

import (
	"fmt"
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func BenchmarkContentString(b *testing.B) {
	content := map[string]any{
		"Files": []provider.RenderItem{},
		"Cover": cover{},
		"Request": provider.Request{
			Path:        "/path/to/file",
			Item:        "file",
			Display:     "grid",
			Preferences: provider.Preferences{},
			Share:       provider.Share{},
			CanEdit:     true,
			CanShare:    false,
			CanWebhook:  false,
		},
		"Message":      "Hello world",
		"HasMap":       true,
		"HasThumbnail": false,
	}

	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%v", content)
	}
}
