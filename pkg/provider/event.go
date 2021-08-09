package provider

import (
	"bytes"
	"errors"
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

var (
	// EventTypeValues string values
	EventTypeValues = []string{"upload", "create", "rename", "delete", "start"}
)

// String return string values
func (et EventType) String() string {
	return EventTypeValues[et]
}

// MarshalJSON marshals the enum as a quoted json string
func (et EventType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(et.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// Event describes an event on fibr
type Event struct {
	Type EventType   `json:"type"`
	Item StorageItem `json:"item"`
	New  StorageItem `json:"new,omitempty"`
}

// NewUploadEvent creates a new upload event
func NewUploadEvent(item StorageItem) Event {
	return Event{
		Type: UploadEvent,
		Item: item,
	}
}

// NewRenameEvent creates a new rename event
func NewRenameEvent(old, new StorageItem) Event {
	return Event{
		Type: RenameEvent,
		Item: old,
		New:  new,
	}
}

// NewDeleteEvent creates a new delete event
func NewDeleteEvent(item StorageItem) Event {
	return Event{
		Type: DeleteEvent,
		Item: item,
	}
}

// NewStartEvent creates a new start event
func NewStartEvent(item StorageItem) Event {
	return Event{
		Type: StartEvent,
		Item: item,
	}
}

// EventBus describes a channel for exchanging Event
type EventBus struct {
	bus  chan Event
	done chan struct{}
}

// NewEventBus create an event exchange channel
func NewEventBus(size int) EventBus {
	return EventBus{
		done: make(chan struct{}),
		bus:  make(chan Event, size),
	}
}

// Push an event in the bus
func (e EventBus) Push(event Event) error {
	select {
	case <-e.done:
		return errors.New("event bus is closed")
	case e.bus <- event:
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
		}
	}()

	<-done
}
