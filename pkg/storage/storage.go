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
		ignore: flags.String(fs, prefix, "crud", "IgnorePattern", "Ignore pattern when listing files or directory", "", nil),
	}
}

func Get(config Config, storage absto.Storage) (absto.Storage, error) {
	ignore := *config.ignore
	if len(ignore) == 0 {
		return storage, nil
	}

	pattern, err := regexp.Compile(ignore)
	if err != nil {
		return storage, fmt.Errorf("regexp compile: %w", err)
	}

	logger.Info("Ignoring files with pattern `%s`", ignore)

	filteredStorage := storage.WithIgnoreFn(func(item absto.Item) bool {
		if strings.HasPrefix(item.Pathname, provider.MetadataDirectoryName) {
			return true
		}

		if pattern.MatchString(item.Name) {
			return true
		}

		return false
	})

	return filteredStorage, nil
}
