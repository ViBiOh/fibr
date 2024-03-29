package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
)

type WebhookKind int

const (
	Raw WebhookKind = iota
	Discord
	Slack
	Telegram
)

var WebhookKindValues = []string{"raw", "discord", "slack", "telegram"}

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

func (w Webhook) Match(e Event) bool {
	if !w.hasType(e.Type) {
		return false
	}

	return w.matchItem(e.Item) || (e.New != nil && w.matchItem(*e.New))
}

func (w Webhook) hasType(eventType EventType) bool {
	for _, t := range w.Types {
		if t == eventType {
			return true
		}
	}

	return false
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
