package version

import (
	"fmt"

	"github.com/ViBiOh/httputils/v4/pkg/sha"
)

var cacheVersion = sha.New("vibioh/fibr/1")[:8]

func Redis(content string) string {
	return fmt.Sprintf("fibr:%s:%s", cacheVersion, content)
}
