package share

import (
	"log/slog"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func (s *Service) PubSubHandle(share provider.Share, err error) {
	if err != nil {
		slog.Error("Share's PubSub", "error", err)
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	slog.Info("Share's PubSub", "id", share.ID)

	if share.Creation.IsZero() {
		delete(s.shares, share.ID)
	} else {
		s.shares[share.ID] = share
	}
}
