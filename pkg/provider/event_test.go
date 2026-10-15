package provider

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestEventBusPushAfterClose(t *testing.T) {
	t.Parallel()

	bus, err := NewEventBus(6, nil, noop.NewTracerProvider())
	if err != nil {
		t.Fatalf("NewEventBus() = %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go bus.Start(ctx, nil, nil)

	cancel()
	<-bus.Done()

	for range 1000 {
		bus.Push(context.Background(), Event{})
	}
}

func TestEventBusDrainsOnClose(t *testing.T) {
	t.Parallel()

	bus, err := NewEventBus(6, nil, noop.NewTracerProvider())
	if err != nil {
		t.Fatalf("NewEventBus() = %s", err)
	}

	var consumed atomic.Int64

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for range 5 {
		bus.Push(context.Background(), Event{})
	}

	go bus.Start(ctx, nil, nil, func(context.Context, Event) {
		consumed.Add(1)
	})

	select {
	case <-bus.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("bus did not stop")
	}

	if got := consumed.Load(); got != 5 {
		t.Errorf("consumed = %d, want 5", got)
	}
}
