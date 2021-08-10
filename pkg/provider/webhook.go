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
	Recursive bool              `json:"recursive"`
}

// Match determine if storage item match webhook
func (w Webhook) Match(item StorageItem) bool {
	if len(item.Name) == 0 {
		return false
	}

	if w.Recursive {
		return strings.HasPrefix(item.Pathname, w.Pathname)
	}

	return item.Dir() == w.Pathname
}
