package main

import (
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httputils"
)

func newPort(clients clients, services services) http.Handler {
	mux := http.NewServeMux()

	services.renderer.RegisterMux(mux, services.fibr.TemplateFunc)

	return httputils.Handler(mux, clients.health,
		clients.telemetry.Middleware("http"),
		services.owasp.Middleware,
	)
}
