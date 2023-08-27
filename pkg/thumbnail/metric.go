package thumbnail

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (s Service) increaseMetric(ctx context.Context, itemType, state string) {
	if s.metric == nil {
		return
	}

	s.metric.Add(ctx, 1, metric.WithAttributes(attribute.String("type", itemType), attribute.String("state", state)))
}
