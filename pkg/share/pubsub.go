package share

import (
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a *App) PubSubHandle(share provider.Share, err error) {
	if err != nil {
		logger.Error("Share's PubSub: %s", err)
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	logger.WithField("id", share.ID).Info("Share's PubSub")

	if share.Creation.IsZero() {
		delete(a.shares, share.ID)
	} else {
		a.shares[share.ID] = share
	}
}
