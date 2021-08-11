package provider

import (
	"strings"
)

// Webhook stores informations about webhook
type Webhook struct {
	Headers   map[string]string `json:"headers,omitempty"`
	ID        string            `json:"id"`
	Pathname  string            `json:"pathname"`
	URL       string            `json:"url"`
	Types     []EventType       `json:"types"`
	Recursive bool              `json:"recursive"`
}

// Match determine if storage item match webhook
func (w Webhook) Match(e Event) bool {
	if !w.hasType(e.Type) {
		return false
	}

	return w.matchItem(e.Item) || (e.New != nil && w.matchItem(*e.New))
}

// Match determine if storage item match webhook
func (w Webhook) hasType(eventType EventType) bool {
	for _, t := range w.Types {
		if t == eventType {
			return true
		}
	}

	return false
}

// Match determine if storage item match webhook
func (w Webhook) matchItem(item StorageItem) bool {
	if len(item.Name) == 0 {
		return false
	}

	if w.Recursive {
		return strings.HasPrefix(item.Pathname, w.Pathname)
	}

	return item.Dir() == w.Pathname
}
