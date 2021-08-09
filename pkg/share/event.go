package share

import (
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

// EventConsumer handle event pushed to the event bus
func (a *App) EventConsumer(e provider.Event) {
	if !a.Enabled() {
		return
	}

	switch e.Type {
	case provider.RenameEvent:
		if err := a.renameItem(e.Item, e.New); err != nil {
			logger.Error("unable to rename share: %s", err)
		}
	case provider.DeleteEvent:
		if err := a.deleteItem(e.Item); err != nil {
			logger.Error("unable to rename share: %s", err)
		}
	}
}

func (a *App) renameItem(old, new provider.StorageItem) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	for id, share := range a.shares {
		if strings.HasPrefix(share.Path, old.Pathname) {
			share.Path = strings.Replace(share.Path, old.Pathname, new.Pathname, 1)
			a.shares[id] = share
		}
	}

	return a.saveShares()
}

func (a *App) deleteItem(item provider.StorageItem) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	for id, share := range a.shares {
		if strings.HasPrefix(share.Path, item.Pathname) {
			delete(a.shares, id)
		}
	}

	return a.saveShares()
}
