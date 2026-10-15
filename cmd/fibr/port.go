package main

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/ViBiOh/httputils/v4/pkg/httputils"
)

func newPort(clients clients, services services) (http.Handler, error) {
	crossOriginProtection, err := newCrossOriginProtection(services.renderer.PublicURL(""))
	if err != nil {
		return nil, fmt.Errorf("cross origin protection: %w", err)
	}

	mux := http.NewServeMux()

	services.renderer.RegisterMux(mux, services.fibr.TemplateFunc)

	return httputils.Handler(
		mux, clients.health,
		clients.telemetry.Middleware("http"),
		services.owasp.Middleware,
		crossOriginProtection.Handler,
	), nil
}

func newCrossOriginProtection(publicURL string) (*http.CrossOriginProtection, error) {
	protection := http.NewCrossOriginProtection()

	parsed, err := url.Parse(publicURL)
	if err != nil {
		return nil, fmt.Errorf("parse public url: %w", err)
	}

	if len(parsed.Scheme) == 0 || len(parsed.Host) == 0 {
		return protection, nil
	}

	if err = protection.AddTrustedOrigin(parsed.Scheme + "://" + parsed.Host); err != nil {
		return nil, fmt.Errorf("add trusted origin: %w", err)
	}

	return protection, nil
}
