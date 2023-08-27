package metadata

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (s Service) increaseExif(ctx context.Context, state string) {
	if s.exifMetric == nil {
		return
	}

	s.exifMetric.Add(ctx, 1, metric.WithAttributes(attribute.String("state", state)))
}

func (s Service) increaseAggregate(ctx context.Context, state string) {
	if s.aggregateMetric == nil {
		return
	}

	s.aggregateMetric.Add(ctx, 1, metric.WithAttributes(attribute.String("state", state)))
}
