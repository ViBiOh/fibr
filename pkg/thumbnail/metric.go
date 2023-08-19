package thumbnail

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (a App) increaseMetric(ctx context.Context, itemType, state string) {
	if a.metric == nil {
		return
	}

	a.metric.Add(ctx, 1, metric.WithAttributes(attribute.String("type", itemType), attribute.String("state", state)))
}
