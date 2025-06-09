package crud

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (s *Service) regenerate(w http.ResponseWriter, r *http.Request, request provider.Request) {
	pathname := request.Filepath()
	ctx := r.Context()

	info, err := s.storage.Stat(ctx, pathname)
	if err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	if !info.IsDir() {
		s.error(w, r, request, model.WrapInvalid(errors.New("regenerate is only available for folder")))
		return
	}

	subset := r.FormValue("subset")

	if len(subset) == 0 {
		s.error(w, r, request, model.WrapInvalid(errors.New("regenerate need a subset")))
		return
	}

	go func(ctx context.Context) {
		var directories []absto.Item

		err := s.storage.Walk(ctx, pathname, func(item absto.Item) error {
			if item.IsDir() {
				directories = append(directories, item)
			} else {
				s.pushEvent(ctx, provider.NewRestartEvent(ctx, item, subset))
			}

			return nil
		})
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "regenerate", slog.String("pathname", pathname), slog.Any("error", err))
		}

		for _, directory := range directories {
			s.pushEvent(ctx, provider.NewStartEvent(ctx, directory))
		}
	}(context.WithoutCancel(ctx))

	s.renderer.Redirect(w, r, "?stats", renderer.NewSuccessMessage("Regeneration of %s in progress...", subset))
}
