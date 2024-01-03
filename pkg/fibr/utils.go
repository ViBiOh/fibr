package fibr

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

func isMethodAllowed(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func SetRouteTag(ctx context.Context, route string) {
	attr := semconv.HTTPRouteKey.String(route)

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attr)

	labeler, _ := otelhttp.LabelerFromContext(ctx)
	labeler.Add(attr)
}
