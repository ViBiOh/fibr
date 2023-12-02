package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type EventType uint

type EventProducer func(context.Context, Event)

type EventConsumer func(context.Context, Event)

type Renamer func(context.Context, absto.Item, absto.Item) error

const (
	UploadEvent EventType = iota
	CreateDir
	RenameEvent
	DeleteEvent
	StartEvent
	AccessEvent
	DescriptionEvent
)

var eventTypeValues = []string{"upload", "create", "rename", "delete", "start", "access", "description"}

func ParseEventType(value string) (EventType, error) {
	for i, eType := range eventTypeValues {
		if strings.EqualFold(eType, value) {
			return EventType(i), nil
		}
	}

	return 0, fmt.Errorf("invalid value `%s` for event type", value)
}

func (et EventType) String() string {
	return eventTypeValues[et]
}

func (et EventType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(et.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

func (et *EventType) UnmarshalJSON(b []byte) error {
	var strValue string
	err := json.Unmarshal(b, &strValue)
	if err != nil {
		return fmt.Errorf("unmarshal event type: %w", err)
	}

	value, err := ParseEventType(strValue)
	if err != nil {
		return fmt.Errorf("parse event type: %w", err)
	}

	*et = value
	return nil
}

type Event struct {
	Time         time.Time         `json:"time"`
	New          *absto.Item       `json:"new,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	URL          string            `json:"url,omitempty"`
	ShareableURL string            `json:"shareable_url,omitempty"`
	TraceLink    trace.Link        `json:"-"`
	Item         absto.Item        `json:"item"`
	Type         EventType         `json:"type"`
}

func (e Event) IsForcedFor(key string) bool {
	force := e.GetMetadata("force")

	return force == "all" || force == key
}

func (e Event) GetMetadata(key string) string {
	if e.Metadata == nil {
		return ""
	}

	return e.Metadata[key]
}

func (e Event) GetURL() string {
	if len(e.ShareableURL) > 0 {
		return e.ShareableURL
	}

	return e.URL
}

func (e Event) BrowserURL() string {
	return e.GetURL() + "?browser"
}

func (e Event) StoryURL(id string) string {
	url := e.GetURL()
	return fmt.Sprintf("%s/?d=story#%s", url[:strings.LastIndex(url, "/")], id)
}

func (e Event) GetName() string {
	if e.New != nil {
		return e.getFrom()
	}

	if e.Item.IsDir() {
		return Dirname(e.Item.Name())
	}

	return e.Item.Name()
}

func (e Event) getFrom() string {
	var fromName string

	if previousDir := path.Dir(e.Item.Pathname); path.Dir(e.New.Pathname) != previousDir {
		fromName = previousDir
	}

	fromName = path.Join(fromName, e.Item.Name())

	if e.Item.IsDir() {
		fromName = Dirname(fromName)
	}

	return fromName
}

func (e Event) GetTo() string {
	if e.New == nil {
		return ""
	}

	var newName string

	if newDir := path.Dir(e.New.Pathname); path.Dir(e.Item.Pathname) != newDir {
		newName = newDir
	}

	newName = path.Join(newName, e.New.Name())

	if e.New.IsDir() {
		newName = Dirname(newName)
	}

	return newName
}

func NewUploadEvent(ctx context.Context, request Request, item absto.Item, shareableURL string, rendererService *renderer.Service) Event {
	return Event{
		Time:         time.Now(),
		Type:         UploadEvent,
		Item:         item,
		TraceLink:    trace.LinkFromContext(ctx),
		URL:          rendererService.PublicURL(request.AbsoluteURL(item.Name())),
		ShareableURL: rendererService.PublicURL(shareableURL),
		Metadata: map[string]string{
			"force": "all",
		},
	}
}

func NewRenameEvent(ctx context.Context, old, new absto.Item, shareableURL string, rendererService *renderer.Service) Event {
	return Event{
		Time:         time.Now(),
		Type:         RenameEvent,
		Item:         old,
		TraceLink:    trace.LinkFromContext(ctx),
		New:          &new,
		URL:          rendererService.PublicURL(new.Pathname),
		ShareableURL: rendererService.PublicURL(shareableURL),
	}
}

func NewDescriptionEvent(ctx context.Context, item absto.Item, shareableURL string, description string, rendererService *renderer.Service) Event {
	return Event{
		Time:         time.Now(),
		Type:         DescriptionEvent,
		Item:         item,
		TraceLink:    trace.LinkFromContext(ctx),
		URL:          rendererService.PublicURL(item.Pathname),
		ShareableURL: rendererService.PublicURL(shareableURL),
		Metadata: map[string]string{
			"description": description,
		},
	}
}

func NewDeleteEvent(ctx context.Context, request Request, item absto.Item, rendererService *renderer.Service) Event {
	return Event{
		Time:      time.Now(),
		Type:      DeleteEvent,
		Item:      item,
		TraceLink: trace.LinkFromContext(ctx),
		URL:       rendererService.PublicURL(request.AbsoluteURL("")),
	}
}

func NewStartEvent(ctx context.Context, item absto.Item) Event {
	return Event{
		Time:      time.Now(),
		Type:      StartEvent,
		Item:      item,
		TraceLink: trace.LinkFromContext(ctx),
	}
}

func NewRestartEvent(ctx context.Context, item absto.Item, subset string) Event {
	return Event{
		Time:      time.Now(),
		Type:      StartEvent,
		Item:      item,
		TraceLink: trace.LinkFromContext(ctx),
		Metadata: map[string]string{
			"force": subset,
		},
	}
}

func NewAccessEvent(ctx context.Context, item absto.Item, r *http.Request) Event {
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
		Time:      time.Now(),
		Type:      AccessEvent,
		Item:      item,
		TraceLink: trace.LinkFromContext(ctx),
		Metadata:  metadata,
		URL:       r.URL.String(),
	}
}

type EventBus struct {
	tracer  trace.Tracer
	counter metric.Int64Counter
	bus     chan Event
	closed  chan struct{}
	done    chan struct{}
}

func NewEventBus(size uint64, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) (EventBus, error) {
	var counter metric.Int64Counter

	if meterProvider != nil {
		meter := meterProvider.Meter("github.com/ViBiOh/fibr/pkr/provider")

		var err error

		counter, err = meter.Int64Counter("fibr.event")
		if err != nil {
			return EventBus{}, fmt.Errorf("create event counter: %w", err)
		}
	}

	return EventBus{
		closed:  make(chan struct{}),
		done:    make(chan struct{}),
		bus:     make(chan Event, size),
		counter: counter,
		tracer:  tracerProvider.Tracer("bus"),
	}, nil
}

func (e EventBus) increaseMetric(ctx context.Context, event Event, state string) {
	if e.counter == nil {
		return
	}

	e.counter.Add(ctx, 1, metric.WithAttributes(attribute.String("type", event.Type.String()), attribute.String("state", state)))
}

func (e EventBus) Done() <-chan struct{} {
	return e.done
}

func (e EventBus) Push(ctx context.Context, event Event) {
	select {
	case <-e.closed:
		e.increaseMetric(ctx, event, "refused")
		slog.ErrorContext(ctx, "bus is closed")
	default:
	}

	select {
	case <-e.closed:
		e.increaseMetric(ctx, event, "refused")
		slog.ErrorContext(ctx, "bus is closed")
	case e.bus <- event:
		e.increaseMetric(ctx, event, "push")
	}
}

func (e EventBus) Start(ctx context.Context, storageService absto.Storage, renamers []Renamer, consumers ...EventConsumer) {
	defer close(e.done)

	go func() {
		defer close(e.bus)
		defer close(e.closed)

		<-ctx.Done()
	}()

	for event := range e.bus {
		ctx, end := telemetry.StartSpan(context.Background(), e.tracer, "event", trace.WithAttributes(attribute.String("type", event.Type.String())), trace.WithLinks(event.TraceLink))

		if event.Type == RenameEvent && event.Item.IsDir() {
			RenameDirectory(ctx, storageService, renamers, event.Item, *event.New)
		}

		for _, consumer := range consumers {
			consumer(ctx, event)
		}

		end(nil)
		e.increaseMetric(ctx, event, "done")
	}
}

func RenameDirectory(ctx context.Context, storageService absto.Storage, renamers []Renamer, old, new absto.Item) {
	if err := storageService.Mkdir(ctx, MetadataDirectory(new), absto.DirectoryPerm); err != nil {
		slog.ErrorContext(ctx, "create new metadata directory", "err", err)
		return
	}

	if err := storageService.Walk(ctx, new.Pathname, func(item absto.Item) error {
		oldItem := item
		oldItem.Pathname = Join(old.Pathname, item.Name())
		oldItem.ID = absto.ID(oldItem.Pathname)

		if item.IsDir() && item.Pathname != new.Pathname {
			RenameDirectory(ctx, storageService, renamers, oldItem, item)
			return nil
		}

		for _, renamer := range renamers {
			if err := renamer(ctx, oldItem, item); err != nil {
				slog.ErrorContext(ctx, "rename metadata", "err", err, "old", oldItem.Pathname, "new", item.Pathname)
			}
		}

		return nil
	}); err != nil {
		slog.ErrorContext(ctx, "walk new metadata directory", "err", err)
	}

	if err := storageService.RemoveAll(ctx, MetadataDirectory(old)); err != nil {
		slog.ErrorContext(ctx, "delete old metadata directory", "err", err)
		return
	}
}
