package version

import (
	"fmt"

	"github.com/ViBiOh/fibr/pkg/provider"
)

var (
	cacheVersion = provider.Hash("vibioh/fibr/4")[:8]
	cachePrefix  = "fibr:" + cacheVersion
)

func Redis(content string) string {
	return fmt.Sprintf("%s:%s", cachePrefix, content)
}
