package exif

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/request"
)

// App of package
type App interface {
	Get(provider.StorageItem) (map[string]interface{}, error)
}

// Config of package
type Config struct {
	exifURL *string
}

type app struct {
	storageApp provider.Storage
	exifURL    string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		exifURL: flags.New(prefix, "exif").Name("ExasURL").Default(flags.Default("ExasURL", "http://exas:1080", overrides)).Label("Exif Tool URL (exas)").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storageApp provider.Storage) App {
	return app{
		exifURL:    strings.TrimSpace(*config.exifURL),
		storageApp: storageApp,
	}
}

// Enabled checks if app is enabled
func (a app) Enabled() bool {
	return len(a.exifURL) != 0
}

// CanHaveExif determine if exif can be extracted for given pathname
func CanHaveExif(item provider.StorageItem) bool {
	return item.IsImage() || item.IsPdf()
}

func (a app) Get(item provider.StorageItem) (map[string]interface{}, error) {
	if !a.Enabled() {
		return nil, nil
	}

	file, err := a.storageApp.ReaderFrom(item.Pathname) // file will be closed by `.Send`
	if err != nil {
		return nil, fmt.Errorf("unable to get reader for `%s`: %s", item.Pathname, err)
	}

	resp, err := request.New().Post(a.exifURL).Send(context.Background(), file)
	if err != nil {
		return nil, err
	}

	var exifs []map[string]interface{}
	if err := httpjson.Read(resp, &exifs); err != nil {
		return nil, fmt.Errorf("unable to read exas response: %s", err)
	}

	if len(exifs) == 0 {
		return nil, nil
	}
	return exifs[0], nil
}
