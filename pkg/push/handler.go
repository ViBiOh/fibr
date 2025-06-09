package push

import (
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (s *Service) Handle(w http.ResponseWriter, r *http.Request, request provider.Request) {
	switch r.Method {
	case http.MethodPost:
		s.post(w, r, request)
	}
}

func (s *Service) post(w http.ResponseWriter, r *http.Request, request provider.Request) {
	ctx := r.Context()

	subscription, err := httpjson.Parse[Subscription](r)
	if err != nil {
		httperror.BadRequest(ctx, w, err)
		return
	}

	item, err := s.storage.Stat(ctx, request.Filepath())
	if err != nil {
		if absto.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		httperror.InternalServerError(ctx, w, err)
		return
	}

	if err := s.Add(ctx, item, subscription); err != nil {
		httperror.InternalServerError(ctx, w, err)
		return
	}
}
