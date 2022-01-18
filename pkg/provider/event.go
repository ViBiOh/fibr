package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/prometheus/client_golang/prometheus"
)

// EventType is the enumeration of event that can happen
type EventType uint

// EventProducer is a func that push an event
type EventProducer func(Event) error

// EventConsumer is a func that consume an event
type EventConsumer func(Event)

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
func NewRenameEvent(old, new absto.Item) Event {
	return Event{
		Time: time.Now(),
		Type: RenameEvent,
		Item: old,
		New:  &new,
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
	counter *prometheus.CounterVec
	bus     chan Event
	done    chan struct{}
}

// NewEventBus create an event exchange channel
func NewEventBus(size uint, prometheusRegisterer prometheus.Registerer) (EventBus, error) {
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
func (e EventBus) Start(done <-chan struct{}, consumers ...EventConsumer) {
	defer close(e.bus)
	defer close(e.done)

	go func() {
		for event := range e.bus {
			for _, consumer := range consumers {
				consumer(event)
			}
			e.increaseMetric(event, "done")
		}
	}()

	<-done
}
