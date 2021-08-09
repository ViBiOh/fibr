package provider

import "strings"

// Webhook stores informations about webhook
type Webhook struct {
	Pathname  string            `json:"pathname"`
	Recursive bool              `json:"recursive"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// Match determine if storage item match webhook
func (w Webhook) Match(item StorageItem) bool {
	if len(item.Pathname) == 0 {
		return false
	}

	if w.Recursive {
		return strings.HasPrefix(item.Pathname, w.Pathname)
	}

	return item.Dir() == w.Pathname
}
