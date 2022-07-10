package provider

import (
	"context"
	"mime/multipart"
	"net/http"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/auth/v2/pkg/ident"
	"github.com/ViBiOh/auth/v2/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

//go:generate mockgen -destination ../mocks/storage.go -mock_names Storage=Storage -package mocks github.com/ViBiOh/absto/pkg/model Storage

// Crud for user to interfact with filesystem
//go:generate mockgen -destination ../mocks/crud.go -mock_names Crud=Crud -package mocks github.com/ViBiOh/fibr/pkg/provider Crud
type Crud interface {
	Start(done <-chan struct{})
	Browser(context.Context, http.ResponseWriter, Request, absto.Item, renderer.Message) (renderer.Page, error)
	List(context.Context, Request, renderer.Message, absto.Item, []absto.Item) (renderer.Page, error)
	Get(http.ResponseWriter, *http.Request, Request) (renderer.Page, error)
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
	Get(string) Share
	Create(context.Context, string, bool, bool, string, bool, time.Duration) (string, error)
	Delete(context.Context, string) error
	List() []Share
}

// WebhookManager description
//go:generate mockgen -destination ../mocks/webhook.go -mock_names WebhookManager=Webhook -package mocks github.com/ViBiOh/fibr/pkg/provider WebhookManager
type WebhookManager interface {
	Create(context.Context, string, bool, WebhookKind, string, []EventType) (string, error)
	Delete(context.Context, string) error
	List() []Webhook
}

// ExifManager description
//go:generate mockgen -destination ../mocks/exif.go -mock_names ExifManager=Exif -package mocks github.com/ViBiOh/fibr/pkg/provider ExifManager
type ExifManager interface {
	ListDir(ctx context.Context, item absto.Item) ([]absto.Item, error)
	GetAggregateFor(ctx context.Context, item absto.Item) (Aggregate, error)
	GetExifFor(ctx context.Context, item absto.Item) (exas.Exif, error)
	SaveExifFor(ctx context.Context, item absto.Item, exif exas.Exif) error
}
