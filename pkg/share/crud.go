package share

import (
	"fmt"
	"path"
	"time"

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

		if _, ok := a.shares[id]; !ok {
			return id, nil
		}
	}
}

// List shares
func (a *App) List() map[string]provider.Share {
	if !a.Enabled() {
		return nil
	}

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.shares
}

// Create a share
func (a *App) Create(filepath string, edit bool, password string, isDir bool, duration time.Duration) (string, error) {
	if !a.Enabled() {
		return "", fmt.Errorf("share is disabled")
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	id, err := a.generateID()
	if err != nil {
		return "", err
	}

	a.shares[id] = provider.Share{
		ID:       id,
		Path:     filepath,
		RootName: path.Base(filepath),
		Edit:     edit,
		Password: password,
		File:     !isDir,
		Creation: a.clock.Now(),
		Duration: duration,
	}

	return id, provider.SaveJSON(a.storageApp, shareFilename, a.shares)
}

// Delete a share
func (a *App) Delete(id string) error {
	if !a.Enabled() {
		return fmt.Errorf("share is disabled")
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	delete(a.shares, id)

	return provider.SaveJSON(a.storageApp, shareFilename, a.shares)
}
