package provider

import (
	"context"
	"net/http"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/auth/v2/pkg/ident"
	"github.com/ViBiOh/auth/v2/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

//go:generate mockgen -destination ../mocks/storage.go -package mocks  -mock_names Storage=Storage github.com/ViBiOh/absto/pkg/model Storage
//go:generate mockgen -source interfaces.go -destination ../mocks/interfaces.go -package mocks -mock_names Crud=Crud,Auth=Auth,ShareManager=ShareManager,WebhookManager=WebhookManager,ExifManager=ExifManager github.com/ViBiOh/absto/pkg/model Storage

type Crud interface {
	Get(http.ResponseWriter, *http.Request, Request) (renderer.Page, error)
	Post(http.ResponseWriter, *http.Request, Request)
	Create(http.ResponseWriter, *http.Request, Request)
	Rename(http.ResponseWriter, *http.Request, Request)
	Delete(http.ResponseWriter, *http.Request, Request)
}

type Auth interface {
	IsAuthenticated(*http.Request) (ident.Provider, model.User, error)
	IsAuthorized(context.Context, string) bool
}

type ShareManager interface {
	Get(string) Share
	Create(context.Context, string, bool, bool, string, bool, time.Duration) (string, error)
	Delete(context.Context, string) error
	List() []Share
}

type WebhookManager interface {
	Create(context.Context, string, bool, WebhookKind, string, []EventType) (string, error)
	Delete(context.Context, string) error
	List() []Webhook
}

type ExifManager interface {
	ListDir(ctx context.Context, item absto.Item) ([]absto.Item, error)
	GetAggregateFor(ctx context.Context, item absto.Item) (Aggregate, error)
	ListAggregateFor(ctx context.Context, items ...absto.Item) (map[string]Aggregate, error)
	SaveAggregateFor(ctx context.Context, item absto.Item, aggregate Aggregate) error
	GetMetadataFor(ctx context.Context, item absto.Item) (Metadata, error)
	ListExifFor(ctx context.Context, items ...absto.Item) (map[string]Metadata, error)
	UpdateDescription(ctx context.Context, item absto.Item, description string) error
}
