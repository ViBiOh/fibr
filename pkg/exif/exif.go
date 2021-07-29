package exif

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/database"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/request"
)

const (
	exifDate = "CreateDate"

	maxThumbnailSize = 1024 * 1024 * 150 // 150mo
)

// App of package
type App interface {
	Get(provider.StorageItem) (map[string]interface{}, error)
	GetDate(provider.StorageItem) (time.Time, error)
	Rename(provider.StorageItem, provider.StorageItem)
	Delete(provider.StorageItem)
}

// Config of package
type Config struct {
	exifURL *string
}

type app struct {
	storageApp  provider.Storage
	databaseApp database.App

	exifURL string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		exifURL: flags.New(prefix, "exif").Name("URL").Default(flags.Default("URL", "", overrides)).Label("Exif Tool URL (exas)").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storageApp provider.Storage, databaseApp database.App) App {
	return app{
		exifURL:     strings.TrimSpace(*config.exifURL),
		storageApp:  storageApp,
		databaseApp: databaseApp,
	}
}

// Enabled checks if app is enabled
func (a app) Enabled() bool {
	return len(a.exifURL) != 0
}

// CanHaveExif determine if exif can be extracted for given pathname
func CanHaveExif(item provider.StorageItem) bool {
	return (item.IsImage() || item.IsPdf()) && item.Size < maxThumbnailSize
}

func (a app) Get(item provider.StorageItem) (map[string]interface{}, error) {
	if !a.Enabled() {
		return nil, nil
	}

	if a.databaseApp.HasEntry([]byte(item.Pathname)) {
		return a.databaseApp.Get([]byte(item.Pathname))
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

	data := exifs[0]

	go func() {
		payload, err := json.Marshal(data)
		if err != nil {
			logger.Error("unable to marshal exif data: %s", err)
		}

		if err := a.databaseApp.Store([]byte(item.Pathname), payload); err != nil {
			logger.Error("unable to store exif in database: %s", err)
		}
	}()

	return data, nil
}

func (a app) GetDate(item provider.StorageItem) (time.Time, error) {
	data, err := a.Get(item)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to retrieve exif data: %s", err)
	}

	if data == nil {
		return time.Time{}, nil
	}

	rawCreateDate, ok := data[exifDate]
	if !ok {
		return time.Time{}, nil
	}

	createDateStr, ok := rawCreateDate.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("key `%s` is not a string", exifDate)
	}

	createDate, err := parseDate(createDateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to parse `%s`: %s", exifDate, err)
	}

	return createDate, nil
}

func parseDate(raw string) (time.Time, error) {
	createDate, err := time.Parse("2006:01:02 15:04:05MST", raw)
	if err == nil {
		return createDate, nil
	}

	createDate, err = time.Parse("2006:01:02 15:04:05-07:00", raw)
	if err == nil {
		return createDate, nil
	}

	createDate, err = time.Parse("2006:01:02 15:04:05Z07:00", raw)
	if err == nil {
		return createDate, nil
	}

	createDate, err = time.Parse("2006:01:02 15:04:05", raw)
	if err == nil {
		return createDate, nil
	}

	createDate, err = time.Parse("01/02/2006 15:04:05", raw)
	if err == nil {
		return createDate, nil
	}

	return time.Time{}, err
}

func (a app) Rename(old, new provider.StorageItem) {
	if err := a.databaseApp.Rename([]byte(old.Pathname), []byte(new.Pathname)); err != nil {
		logger.Error("unable to rename exif: %s", err)
	}
}

func (a app) Delete(item provider.StorageItem) {
	if err := a.databaseApp.Delete([]byte(item.Pathname)); err != nil {
		logger.Error("unable to delete exif: %s", err)
	}
}
