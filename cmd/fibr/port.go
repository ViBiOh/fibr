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
	handler   http.Handler
	name      string
	serverApp server.App
}

type ports []port

func newPorts(config configuration, clients client, services services) ports {
	return ports{
		{
			serverApp: server.New(config.appServer),
			name:      "http",
			handler: httputils.Handler(
				services.rendererApp.Handler(services.fibrApp.TemplateFunc),
				clients.health, recoverer.Middleware, clients.prometheus.Middleware, clients.tracer.Middleware, owasp.New(config.owasp).Middleware,
			),
		},
		{
			serverApp: server.New(config.promServer),
			name:      "prometheus",
			handler:   clients.prometheus.Handler(),
		},
	}
}

func (p ports) Start(ctx context.Context) {
	for _, instance := range p {
		go instance.serverApp.Start(ctx, instance.name, instance.handler)
	}
}

func (p ports) TerminateOnDone() <-chan struct{} {
	return p[0].serverApp.Done()
}

func (p ports) GracefulWait() {
	for _, instance := range p {
		<-instance.serverApp.Done()
	}
}
