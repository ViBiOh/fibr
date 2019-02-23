package minio

import (
	"flag"
	"fmt"
	"io"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/errors"
	"github.com/ViBiOh/httputils/pkg/tools"
	"github.com/minio/minio-go"
)

// Config of package
type Config struct {
	endpoint  *string
	accessKey *string
	secretKey *string
	useSSL    *bool
}

// App of package
type App struct {
	client *minio.Client
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		endpoint:  fs.String(tools.ToCamel(fmt.Sprintf(`%sEndpoint`, prefix)), ``, fmt.Sprintf(`[%s] Endpoint`, prefix)),
		accessKey: fs.String(tools.ToCamel(fmt.Sprintf(`%sAccessKey`, prefix)), ``, fmt.Sprintf(`[%s] Access key`, prefix)),
		secretKey: fs.String(tools.ToCamel(fmt.Sprintf(`%sSecretKey`, prefix)), ``, fmt.Sprintf(`[%s] Secret key`, prefix)),
		useSSL:    fs.Bool(tools.ToCamel(fmt.Sprintf(`%sSsl`, prefix)), true, fmt.Sprintf(`[%s] Use SSL`, prefix)),
	}
}

// New creates new App from Config
func New(config Config) (*App, error) {
	if *config.endpoint == `` {
		return nil, errors.New(`no endpoint provided`)
	}

	minioClient, err := minio.New(*config.endpoint, *config.accessKey, *config.secretKey, *config.useSSL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &App{
		client: minioClient,
	}, nil
}

// Name of the storage
func (a App) Name() string {
	return `minio`
}

// Root name of the storage
func (a App) Root() string {
	return ``
}

// Info provide metadata about given pathname
func (a App) Info(pathname string) (*provider.StorageItem, error) {
	return nil, nil
}

// Open writer for given pathname
func (a App) Open(pathname string) (io.WriteCloser, error) {
	return nil, nil
}

// Read content of given pathname
func (a App) Read(pathname string) (io.ReadCloser, error) {
	return nil, nil
}

// Serve file for given pathname
func (a App) Serve(http.ResponseWriter, *http.Request, string) {
}

// List item in the storage
func (a App) List(pathname string) ([]*provider.StorageItem, error) {
	return nil, nil
}

// Walk browse item recursively
func (a App) Walk(walkFn func(string, *provider.StorageItem, error) error) error {
	return nil
}

// Create container in storage
func (a App) Create(name string) error {
	return nil
}

// Upload file to storage
func (a App) Upload(pathname string, content io.ReadCloser) error {
	return nil
}

// Rename file or directory from storage
func (a App) Rename(oldName, newName string) error {
	return nil
}

// Remove file or directory from storage
func (a App) Remove(pathname string) error {
	return nil
}
