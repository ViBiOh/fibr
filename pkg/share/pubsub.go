package share

import (
	"log/slog"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func (a *App) PubSubHandle(share provider.Share, err error) {
	if err != nil {
		slog.Error("Share's PubSub", "err", err)
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	slog.Info("Share's PubSub", "id", share.ID)

	if share.Creation.IsZero() {
		delete(a.shares, share.ID)
	} else {
		a.shares[share.ID] = share
	}
}
