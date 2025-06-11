package push

import (
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (s *Service) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.post(w, r)
	}
}

func (s *Service) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	subscription, err := httpjson.Parse[Subscription](r)
	if err != nil {
		httperror.BadRequest(ctx, w, err)
		return
	}

	if err := s.Add(ctx, subscription); err != nil {
		httperror.InternalServerError(ctx, w, err)
		return
	}
}
