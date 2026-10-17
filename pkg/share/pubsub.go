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

	s.shares.Patch(func(shares map[string]provider.Share) {
		if share.Created.IsZero() {
			delete(shares, share.ID)
		} else {
			shares[share.ID] = share
		}
	})
}
