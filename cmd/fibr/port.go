package main

import (
	"net/http"
)

func newPort(services services) http.Handler {
	mux := http.NewServeMux()

	services.renderer.Register(mux, services.fibr.TemplateFunc)

	return mux
}
