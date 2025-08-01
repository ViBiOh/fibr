package share

import (
	"context"
	"log/slog"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func (s *Service) PubSubHandle(share provider.Share, err error) {
	if err != nil {
		slog.LogAttrs(context.Background(), slog.LevelError, "Share's PubSub", slog.Any("error", err))
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if share.Created.IsZero() {
		delete(s.shares, share.ID)
	} else {
		s.shares[share.ID] = share
	}
}
