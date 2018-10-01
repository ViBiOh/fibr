package minio

import (
	"flag"
	"fmt"
	"strings"

	"github.com/ViBiOh/httputils/pkg/tools"
	miniolib "github.com/minio/minio-go"
)

// App stores informations
type App struct {
	client *miniolib.Client
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]*string) (*App, error) {
	endpoint := strings.TrimSpace(*config[`endpoint`])
	accessKey := strings.TrimSpace(*config[`accessKey`])
	secretKey := strings.TrimSpace(*config[`secretKey`])
	useSSL := true

	minioClient, err := miniolib.New(endpoint, accessKey, secretKey, useSSL)
	if err != nil {
		return nil, fmt.Errorf(`error while initializing Minio client:  %v`, err)
	}

	return &App{
		client: minioClient,
	}, nil
}

// Flags adds flags for given prefix
func Flags(prefix string) map[string]*string {
	return map[string]*string{
		`endpoint`:  flag.String(tools.ToCamel(fmt.Sprintf(`%sEndpoint`, prefix)), ``, `[minio] Endpoint server`),
		`accessKey`: flag.String(tools.ToCamel(fmt.Sprintf(`%sAccessKey`, prefix)), ``, `[minio] Access Key`),
		`secretKey`: flag.String(tools.ToCamel(fmt.Sprintf(`%sSecretKey`, prefix)), ``, `[minio] Secret Key`),
	}
}
