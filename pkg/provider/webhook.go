package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
)

type WebhookKind int

const (
	Raw WebhookKind = iota
	Discord
	Slack
	Telegram
	Push
)

var WebhookKindValues = []string{"raw", "discord", "slack", "telegram", "push"}

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

func (r WebhookKind) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(r.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

func (r *WebhookKind) UnmarshalJSON(b []byte) error {
	var strValue string
	err := json.Unmarshal(b, &strValue)
	if err != nil {
		return fmt.Errorf("unmarshal event type: %w", err)
	}

	value, err := ParseWebhookKind(strValue)
	if err != nil {
		return fmt.Errorf("parse event type: %w", err)
	}

	*r = value
	return nil
}

type Webhook struct {
	ID        string      `json:"id"`
	Pathname  string      `json:"pathname"`
	URL       string      `json:"url"`
	Types     []EventType `json:"types"`
	Kind      WebhookKind `json:"kind"`
	Recursive bool        `json:"recursive"`
}

func (w Webhook) Similar(other Webhook) bool {
	if w.Pathname != other.Pathname || w.Kind != other.Kind || w.URL != other.URL || w.Recursive != other.Recursive {
		return false
	}

	if len(w.Types) != len(other.Types) {
		return false
	}

	for _, eventType := range w.Types {
		if !slices.Contains(other.Types, eventType) {
			return false
		}
	}

	return true
}

func (w Webhook) Match(e Event) bool {
	if !w.hasType(e.Type) {
		return false
	}

	return w.matchItem(e.Item) || (e.New != nil && w.matchItem(*e.New))
}

func (w Webhook) hasType(eventType EventType) bool {
	return slices.Contains(w.Types, eventType)
}

func (w Webhook) matchItem(item absto.Item) bool {
	if item.IsZero() {
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
