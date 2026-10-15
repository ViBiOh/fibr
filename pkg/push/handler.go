package push

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

const maxSubscriptionBodySize = 8 << 10

func (s *Service) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	r.Body = http.MaxBytesReader(w, r.Body, maxSubscriptionBodySize)

	subscription, err := httpjson.Parse[Subscription](r)
	if err != nil {
		httperror.BadRequest(ctx, w, err)
		return
	}

	if err = s.validate(subscription); err != nil {
		httperror.BadRequest(ctx, w, err)
		return
	}

	subscription.Created = time.Now()

	if err = s.Add(ctx, subscription); err != nil {
		if errors.Is(err, ErrTooManySubscriptions) {
			slog.LogAttrs(ctx, slog.LevelWarn, "push subscription refused", slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusTooManyRequests)

			return
		}

		httperror.InternalServerError(ctx, w, err)
	}
}
