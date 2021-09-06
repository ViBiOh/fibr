package s3

import (
	"context"
	"flag"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/flags"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// App of package
type App struct {
	ignoreFn func(provider.StorageItem) bool
	bucket   string
	client   *minio.Client
}

// Config of package
type Config struct {
	endpoint     *string
	accessKey    *string
	secretAccess *string
	bucket       *string
	useSSL       *bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		endpoint:     flags.New(prefix, "s3", "Endpoint").Default("", overrides).Label("Storage Object endpoint").ToString(fs),
		accessKey:    flags.New(prefix, "s3", "AccessKey").Default("", overrides).Label("Storage Object Access Key").ToString(fs),
		secretAccess: flags.New(prefix, "s3", "SecretAccess").Default("", overrides).Label("Storage Object Secret Access").ToString(fs),
		bucket:       flags.New(prefix, "s3", "Bucket").Default("", overrides).Label("Storage Object Bucket").ToString(fs),
		useSSL:       flags.New(prefix, "s3", "SSL").Default(true, overrides).Label("Use SSL").ToBool(fs),
	}
}

// New creates new App from Config
func New(config Config) (App, error) {
	if len(*config.endpoint) == 0 {
		return App{}, nil
	}

	client, err := minio.New(*config.endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(*config.accessKey, *config.secretAccess, ""),
		Secure: *config.useSSL,
	})
	if err != nil {
		return App{}, fmt.Errorf("unable to create minio client: %s", err)
	}

	return App{
		client: client,
		bucket: *config.bucket,
	}, nil
}

// Enabled checks that requirements are met
func (a App) Enabled() bool {
	return a.client != nil
}

// WithIgnoreFn create a new App with given ignoreFn
func (a App) WithIgnoreFn(ignoreFn func(provider.StorageItem) bool) provider.Storage {
	a.ignoreFn = ignoreFn

	return a
}

// Info provide metadata about given pathname
func (a App) Info(pathname string) (provider.StorageItem, error) {
	if len(getPath(pathname)) == 0 {
		return provider.StorageItem{
			Name:     "root",
			Pathname: "",
			IsDir:    true,
		}, nil
	}

	info, err := a.client.StatObject(context.Background(), a.bucket, getPath(pathname), minio.GetObjectOptions{})
	if err != nil {
		return provider.StorageItem{}, fmt.Errorf("unable to retrieve object: %s", err)
	}

	return convertToItem(pathname, info), nil
}

// List items in the storage
func (a App) List(pathname string) ([]provider.StorageItem, error) {
	objectsCh := a.client.ListObjects(context.Background(), a.bucket, minio.ListObjectsOptions{
		Prefix: getPath(pathname),
	})

	var items []provider.StorageItem
	for object := range objectsCh {
		item := convertToItem(pathname, object)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			continue
		}

		items = append(items, item)
	}

	sort.Sort(provider.ByHybridSort(items))

	return items, nil
}

// WriterTo opens writer for given pathname
func (a App) WriterTo(pathname string) (io.WriteCloser, error) {
	reader, writer := io.Pipe()

	if _, err := a.client.PutObject(context.Background(), a.bucket, getPath(pathname), reader, -1, minio.PutObjectOptions{}); err != nil {
		err = fmt.Errorf("unable to put object: %s", err)

		if closeErr := writer.Close(); closeErr != nil {
			err = fmt.Errorf("%s: %w", err, closeErr)
		}

		return nil, err
	}

	return writer, nil
}

// ReaderFrom reads content from given pathname
func (a App) ReaderFrom(pathname string) (provider.StorageReader, error) {
	object, err := a.client.GetObject(context.Background(), a.bucket, getPath(pathname), minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get object: %s", err)
	}

	return object, nil
}

// UpdateDate update date from given value
func (a App) UpdateDate(pathname string, date time.Time) error {
	// TODO

	return nil
}

// Walk browses item recursively
func (a App) Walk(pathname string, walkFn func(provider.StorageItem, error) error) error {
	objectsCh := a.client.ListObjects(context.Background(), a.bucket, minio.ListObjectsOptions{
		Prefix:    getPath(pathname),
		Recursive: true,
	})

	for object := range objectsCh {
		item := convertToItem(pathname, object)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			continue
		}

		if err := walkFn(item, nil); err != nil {
			return err
		}
	}

	return nil
}

// CreateDir container in storage
func (a App) CreateDir(name string) error {
	// TODO

	return nil
}

// Rename file or directory from storage
func (a App) Rename(oldName, newName string) error {
	// TODO

	return nil
}

// Remove file or directory from storage
func (a App) Remove(pathname string) error {
	// TODO

	return nil
}
