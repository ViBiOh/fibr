package main

import (
	"context"
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/server"
)

type port struct {
	handler http.Handler
	name    string
	server  server.Server
}

type ports []port

func newPorts(config configuration, clients client, services services) ports {
	return ports{
		{
			server: server.New(config.appServer),
			name:   "http",
			handler: httputils.Handler(
				services.renderer.Handler(services.fibr.TemplateFunc),
				clients.health, recoverer.Middleware, clients.telemetry.Middleware("http"), owasp.New(config.owasp).Middleware,
			),
		},
	}
}

func (p ports) Start(ctx context.Context) {
	for _, instance := range p {
		go instance.server.Start(ctx, instance.name, instance.handler)
	}
}

func (p ports) TerminateOnDone() <-chan struct{} {
	return p[0].server.Done()
}

func (p ports) GracefulWait() {
	for _, instance := range p {
		<-instance.server.Done()
	}
}
