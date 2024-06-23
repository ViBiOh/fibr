package main

import (
	"net/http"
)

func newPort(config configuration, services services) http.Handler {
	mux := http.NewServeMux()

	mux.Handle(config.renderer.PathPrefix+"/", services.renderer.NewServeMux(services.fibr.TemplateFunc))

	return mux
}
