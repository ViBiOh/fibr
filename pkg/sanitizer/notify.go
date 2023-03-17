package sanitizer

import (
	"context"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"go.opentelemetry.io/otel/trace"
)

func (a App) notify(ctx context.Context, event provider.Event) {
	if a.pushEvent == nil {
		return
	}

	event.TraceLink = trace.LinkFromContext(ctx)

	if err := a.pushEvent(event); err != nil {
		logger.Error("push event %+v: %s", event, err)
	}
}
