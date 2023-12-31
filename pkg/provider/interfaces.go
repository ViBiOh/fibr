package provider

import (
	"context"
	"net/http"
	"time"

	"github.com/ViBiOh/auth/v2/pkg/ident"
	"github.com/ViBiOh/auth/v2/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

//go:generate mockgen -destination ../mocks/storage.go -package mocks -mock_names Storage=Storage github.com/ViBiOh/absto/pkg/model Storage
//go:generate mockgen -destination ../mocks/redis_client.go -package mocks -mock_names Client=RedisClient github.com/ViBiOh/httputils/v4/pkg/redis Client

//go:generate mockgen -source interfaces.go -destination ../mocks/interfaces.go -package mocks -mock_names Crud=Crud,Auth=Auth,ShareManager=ShareManager,WebhookManager=WebhookManager

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
	List() []Share
	Get(string) Share
	Create(context.Context, string, bool, bool, string, bool, time.Duration) (string, error)
	UpdatePassword(context.Context, string, string) error
	Delete(context.Context, string) error
}

type WebhookManager interface {
	List() []Webhook
	Create(context.Context, string, bool, WebhookKind, string, []EventType) (string, error)
	Delete(context.Context, string) error
}
