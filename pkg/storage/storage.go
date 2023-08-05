package storage

import (
	"flag"
	"fmt"
	"regexp"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

type Config struct {
	ignore *string
}

func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		ignore: flags.New("IgnorePattern", "Ignore pattern when listing files or directory").Prefix(prefix).DocPrefix("crud").String(fs, "", nil),
	}
}

func Get(config Config, storage absto.Storage) (absto.Storage, error) {
	var pattern *regexp.Regexp

	if ignore := *config.ignore; len(ignore) != 0 {
		var err error

		pattern, err = regexp.Compile(ignore)
		if err != nil {
			return storage, fmt.Errorf("regexp compile: %w", err)
		}

		logger.Info("Ignoring files with pattern `%s`", ignore)
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
