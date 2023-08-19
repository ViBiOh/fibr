package metadata

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (a App) increaseExif(ctx context.Context, state string) {
	if a.exifMetric == nil {
		return
	}

	a.exifMetric.Add(ctx, 1, metric.WithAttributes(attribute.String("state", state)))
}

func (a App) increaseAggregate(ctx context.Context, state string) {
	if a.aggregateMetric == nil {
		return
	}

	a.aggregateMetric.Add(ctx, 1, metric.WithAttributes(attribute.String("state", state)))
}
