package storage

import (
	"flag"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
)

type Config struct {
	Ignore string
}

func Flags(fs *flag.FlagSet, prefix string) *Config {
	var config Config

	flags.New("IgnorePattern", "Ignore pattern when listing files or directory").Prefix(prefix).DocPrefix("crud").StringVar(fs, &config.Ignore, "", nil)

	return &config
}

func Get(config *Config, storage absto.Storage) (absto.Storage, error) {
	var pattern *regexp.Regexp

	if len(config.Ignore) != 0 {
		var err error

		pattern, err = regexp.Compile(config.Ignore)
		if err != nil {
			return storage, fmt.Errorf("regexp compile: %w", err)
		}

		slog.Info("Ignoring files with pattern", "pattern", config.Ignore)
	}

	return storage.WithIgnoreFn(func(item absto.Item) bool {
		if strings.HasPrefix(item.Pathname, provider.MetadataDirectoryName) {
			return true
		}

		if pattern != nil && pattern.MatchString(item.Name()) {
			return true
		}

		return false
	}), nil
}
