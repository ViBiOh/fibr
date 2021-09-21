package provider

import (
	"context"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/ViBiOh/auth/v2/pkg/ident"
	"github.com/ViBiOh/auth/v2/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// Crud for user to interfact with filesystem
//go:generate mockgen -destination ../mocks/crud.go -mock_names Crud=Crud -package mocks github.com/ViBiOh/fibr/pkg/provider Crud
type Crud interface {
	Start(done <-chan struct{})
	Browser(http.ResponseWriter, Request, StorageItem, renderer.Message) (string, int, map[string]interface{}, error)
	List(http.ResponseWriter, Request, renderer.Message) (string, int, map[string]interface{}, error)
	Get(http.ResponseWriter, *http.Request, Request) (string, int, map[string]interface{}, error)
	Post(http.ResponseWriter, *http.Request, Request)
	Create(http.ResponseWriter, *http.Request, Request)
	Upload(http.ResponseWriter, *http.Request, Request, map[string]string, *multipart.Part)
	Rename(http.ResponseWriter, *http.Request, Request)
	Delete(http.ResponseWriter, *http.Request, Request)
}

// Auth manager user authentication/authorization
//go:generate mockgen -destination ../mocks/auth.go -mock_names Auth=Auth -package mocks github.com/ViBiOh/fibr/pkg/provider Auth
type Auth interface {
	IsAuthenticated(*http.Request) (ident.Provider, model.User, error)
	IsAuthorized(context.Context, string) bool
}

// ShareManager description
//go:generate mockgen -destination ../mocks/share.go -mock_names ShareManager=Share -package mocks github.com/ViBiOh/fibr/pkg/provider ShareManager
type ShareManager interface {
	Enabled() bool
	Get(string) Share
	Create(string, bool, string, bool, time.Duration) (string, error)
	Delete(string) error
	List() map[string]Share
}

// WebhookManager description
//go:generate mockgen -destination ../mocks/webhook.go -mock_names WebhookManager=Webhook -package mocks github.com/ViBiOh/fibr/pkg/provider WebhookManager
type WebhookManager interface {
	Enabled() bool
	Create(string, bool, string, []EventType) (string, error)
	Delete(string) error
	List() map[string]Webhook
}
