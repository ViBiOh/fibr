package webhook

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (a *App) increaseMetric(ctx context.Context, code string) {
	if a.counter == nil {
		return
	}

	a.counter.Add(ctx, 1, metric.WithAttributes(attribute.String("code", code)))
}
