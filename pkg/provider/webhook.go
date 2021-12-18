package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// WebhookKind defines constant for webhook kind
type WebhookKind int

const (
	// Raw webhook
	Raw WebhookKind = iota
	// Discord webhook
	Discord
	// Slack webhook
	Slack
	// Telegram webhook
	Telegram
)

// WebhookKindValues string values
var WebhookKindValues = []string{"raw", "discord", "slack", "telegram"}

// ParseWebhookKind parse raw string into a WebhookKind
func ParseWebhookKind(value string) (WebhookKind, error) {
	for i, short := range WebhookKindValues {
		if strings.EqualFold(short, value) {
			return WebhookKind(i), nil
		}
	}

	return Raw, fmt.Errorf("invalid value `%s` for webhook kind", value)
}

func (r WebhookKind) String() string {
	return WebhookKindValues[r]
}

// MarshalJSON marshals the enum as a quoted json string
func (r WebhookKind) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(r.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmarshal JSON
func (r *WebhookKind) UnmarshalJSON(b []byte) error {
	var strValue string
	err := json.Unmarshal(b, &strValue)
	if err != nil {
		return fmt.Errorf("unable to unmarshal event type: %s", err)
	}

	value, err := ParseWebhookKind(strValue)
	if err != nil {
		return fmt.Errorf("unable to parse event type: %s", err)
	}

	*r = value
	return nil
}

// Webhook stores informations about webhook
type Webhook struct {
	ID        string      `json:"id"`
	Pathname  string      `json:"pathname"`
	URL       string      `json:"url"`
	Types     []EventType `json:"types"`
	Kind      WebhookKind `json:"kind"`
	Recursive bool        `json:"recursive"`
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

	itemDir := item.Dir()

	if len(w.Pathname) == 0 {
		return len(itemDir) == 0 || itemDir == "/"
	}

	return itemDir == w.Pathname
}
