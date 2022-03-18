package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// EventType is the enumeration of event that can happen
type EventType uint

// EventProducer is a func that push an event
type EventProducer func(Event) error

// EventConsumer is a func that consume an event
type EventConsumer func(context.Context, Event)

// Renamer is a func that rename an item
type Renamer func(context.Context, absto.Item, absto.Item) error

const (
	// UploadEvent occurs when someone upload a file
	UploadEvent EventType = iota
	// CreateDir occurs when a directory is created
	CreateDir
	// RenameEvent occurs when an item is renamed
	RenameEvent
	// DeleteEvent occurs when an item is deleted
	DeleteEvent
	// StartEvent occurs when fibr start
	StartEvent
	// AccessEvent occurs when content is accessed
	AccessEvent
)

var eventTypeValues = []string{"upload", "create", "rename", "delete", "start", "access"}

// ParseEventType parse raw string into an EventType
func ParseEventType(value string) (EventType, error) {
	for i, eType := range eventTypeValues {
		if strings.EqualFold(eType, value) {
			return EventType(i), nil
		}
	}

	return 0, fmt.Errorf("invalid value `%s` for event type", value)
}

// String return string values
func (et EventType) String() string {
	return eventTypeValues[et]
}

// MarshalJSON marshals the enum as a quoted json string
func (et EventType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(et.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmarshal JSON
func (et *EventType) UnmarshalJSON(b []byte) error {
	var strValue string
	err := json.Unmarshal(b, &strValue)
	if err != nil {
		return fmt.Errorf("unable to unmarshal event type: %s", err)
	}

	value, err := ParseEventType(strValue)
	if err != nil {
		return fmt.Errorf("unable to parse event type: %s", err)
	}

	*et = value
	return nil
}

// Event describes an event on fibr
type Event struct {
	Time         time.Time         `json:"time"`
	New          *absto.Item       `json:"new,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	URL          string            `json:"url,omitempty"`
	ShareableURL string            `json:"shareable_url,omitempty"`
	Item         absto.Item        `json:"item"`
	Type         EventType         `json:"type"`
}

// IsForcedFor check if event is forced for given key
func (e Event) IsForcedFor(key string) bool {
	force := e.GetMetadata("force")

	return force == "all" || force == key
}

// GetMetadata extracts metadata content
func (e Event) GetMetadata(key string) string {
	if e.Metadata == nil {
		return ""
	}

	return e.Metadata[key]
}

// GetURL returns the appropriate URL for the event
func (e Event) GetURL() string {
	if len(e.ShareableURL) > 0 {
		return e.ShareableURL
	}

	return e.URL
}

// GetName returns the name of the item
func (e Event) GetName() string {
	if e.New != nil {
		return e.getFrom()
	}

	if e.Item.IsDir {
		return Dirname(e.Item.Name)
	}

	return e.Item.Name
}

func (e Event) getFrom() string {
	var fromName string

	if previousDir := path.Dir(e.Item.Pathname); path.Dir(e.New.Pathname) != previousDir {
		fromName = previousDir
	}

	fromName = path.Join(fromName, e.Item.Name)

	if e.Item.IsDir {
		fromName = Dirname(fromName)
	}

	return fromName
}

// GetTo returns the appropriate to destination
func (e Event) GetTo() string {
	if e.New == nil {
		return ""
	}

	var newName string

	if newDir := path.Dir(e.New.Pathname); path.Dir(e.Item.Pathname) != newDir {
		newName = newDir
	}

	newName = path.Join(newName, e.New.Name)

	if e.New.IsDir {
		newName = Dirname(newName)
	}

	return newName
}

// NewUploadEvent creates a new upload event
func NewUploadEvent(request Request, item absto.Item, shareableURL string, rendererApp renderer.App) Event {
	return Event{
		Time:         time.Now(),
		Type:         UploadEvent,
		Item:         item,
		URL:          rendererApp.PublicURL(request.AbsoluteURL(item.Name)),
		ShareableURL: rendererApp.PublicURL(shareableURL),
	}
}

// NewRenameEvent creates a new rename event
func NewRenameEvent(old, new absto.Item, shareableURL string, rendererApp renderer.App) Event {
	return Event{
		Time:         time.Now(),
		Type:         RenameEvent,
		Item:         old,
		New:          &new,
		URL:          rendererApp.PublicURL(new.Pathname),
		ShareableURL: rendererApp.PublicURL(shareableURL),
	}
}

// NewDeleteEvent creates a new delete event
func NewDeleteEvent(request Request, item absto.Item, rendererApp renderer.App) Event {
	return Event{
		Time: time.Now(),
		Type: DeleteEvent,
		Item: item,
		URL:  rendererApp.PublicURL(request.AbsoluteURL("")),
	}
}

// NewStartEvent creates a new start event
func NewStartEvent(item absto.Item) Event {
	return Event{
		Time: time.Now(),
		Type: StartEvent,
		Item: item,
	}
}

// NewRestartEvent creates a new restart event
func NewRestartEvent(item absto.Item, subset string) Event {
	return Event{
		Time: time.Now(),
		Type: StartEvent,
		Item: item,
		Metadata: map[string]string{
			"force": subset,
		},
	}
}

// NewAccessEvent creates a new access event
func NewAccessEvent(item absto.Item, r *http.Request) Event {
	metadata := make(map[string]string)
	for key, values := range r.Header {
		if strings.EqualFold(key, "Authorization") {
			continue
		}

		metadata[key] = strings.Join(values, ", ")
	}

	metadata["Method"] = r.Method
	metadata["URL"] = r.URL.String()

	return Event{
		Time:     time.Now(),
		Type:     AccessEvent,
		Item:     item,
		Metadata: metadata,
		URL:      r.URL.String(),
	}
}

// EventBus describes a channel for exchanging Event
type EventBus struct {
	tracer  trace.Tracer
	counter *prometheus.CounterVec
	bus     chan Event
	done    chan struct{}
}

// NewEventBus create an event exchange channel
func NewEventBus(size uint, prometheusRegisterer prometheus.Registerer, tracerApp tracer.App) (EventBus, error) {
	var counter *prometheus.CounterVec

	if prometheusRegisterer != nil {
		counter = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "fibr",
			Subsystem: "event",
			Name:      "item",
		}, []string{"type", "state"})

		if err := prometheusRegisterer.Register(counter); err != nil {
			return EventBus{}, fmt.Errorf("unable to register event metric: %s", err)
		}
	}

	return EventBus{
		done:    make(chan struct{}),
		bus:     make(chan Event, size),
		counter: counter,
		tracer:  tracerApp.GetTracer("event"),
	}, nil
}

func (e EventBus) increaseMetric(event Event, state string) {
	if e.counter == nil {
		return
	}

	e.counter.WithLabelValues(event.Type.String(), state).Inc()
}

// Push an event in the bus
func (e EventBus) Push(event Event) error {
	select {
	case <-e.done:
		e.increaseMetric(event, "refused")
		return errors.New("done signal is received")
	case e.bus <- event:
		e.increaseMetric(event, "push")
		return nil
	}
}

// Start the distibution of Event
func (e EventBus) Start(done <-chan struct{}, storageApp absto.Storage, renamers []Renamer, consumers ...EventConsumer) {
	defer close(e.bus)
	defer close(e.done)

	go func() {
		for event := range e.bus {
			var span trace.Span
			ctx := context.Background()

			if e.tracer != nil {
				ctx, span = e.tracer.Start(ctx, "event")
				span.SetAttributes(attribute.String("type", event.Type.String()))
			}

			if event.Type == RenameEvent && event.Item.IsDir {
				RenameDirectory(ctx, storageApp, renamers, event.Item, *event.New)
			}

			for _, consumer := range consumers {
				consumer(ctx, event)
			}

			if span != nil {
				span.End()
			}
			e.increaseMetric(event, "done")
		}
	}()

	<-done
}

// RenameDirectory for metadata
func RenameDirectory(ctx context.Context, storageApp absto.Storage, renamers []Renamer, old, new absto.Item) {
	if err := storageApp.CreateDir(ctx, MetadataDirectory(new)); err != nil {
		logger.Error("unable to create new metadata directory: %s", err)
		return
	}

	if err := storageApp.Walk(ctx, new.Pathname, func(item absto.Item) error {
		oldItem := item
		oldItem.Pathname = Join(old.Pathname, item.Name)
		oldItem.ID = absto.ID(oldItem.Pathname)

		if item.IsDir && item.Pathname != new.Pathname {
			RenameDirectory(ctx, storageApp, renamers, oldItem, item)
			return nil
		}

		for _, renamer := range renamers {
			if err := renamer(ctx, oldItem, item); err != nil {
				logger.Error("unable to rename metadata from `%s` to `%s`: %s", oldItem.Pathname, item.Pathname, err)
			}
		}

		return nil
	}); err != nil {
		logger.Error("unable to walk new metadata directory: %s", err)
	}

	if err := storageApp.Remove(ctx, MetadataDirectory(old)); err != nil {
		logger.Error("unable to delete old metadata directory: %s", err)
		return
	}
}
