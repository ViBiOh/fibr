package webhook

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (s *Service) increaseMetric(ctx context.Context, code string) {
	if s.counter == nil {
		return
	}

	s.counter.Add(ctx, 1, metric.WithAttributes(attribute.String("code", code)))
}
