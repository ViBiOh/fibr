package webhook

import (
	"fmt"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/httputils/v4/pkg/uuid"
)

func (a *App) generateID() (string, error) {
	for {
		uuid, err := uuid.New()
		if err != nil {
			return "", err
		}
		id := sha.New(uuid)[:8]

		if _, ok := a.webhooks[id]; !ok {
			return id, nil
		}
	}
}

// List webhooks
func (a *App) List() map[string]provider.Webhook {
	if !a.Enabled() {
		return nil
	}

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.webhooks
}

// Create a webhook
func (a *App) Create(pathname string, recursive bool, url string, types []provider.EventType) (string, error) {
	if !a.Enabled() {
		return "", fmt.Errorf("webhook is disabled")
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	id, err := a.generateID()
	if err != nil {
		return "", err
	}

	a.webhooks[id] = provider.Webhook{
		ID:        id,
		Pathname:  pathname,
		Recursive: recursive,
		URL:       url,
		Types:     types,
	}

	return id, a.saveWebhooks()
}

// Delete a webhook
func (a *App) Delete(id string) error {
	if !a.Enabled() {
		return fmt.Errorf("webhook is disabled")
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	delete(a.webhooks, id)

	return a.saveWebhooks()
}
