package crud

import (
	"context"
	"errors"
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/cntxt"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a App) regenerate(w http.ResponseWriter, r *http.Request, request provider.Request) {
	pathname := request.Filepath()
	ctx := r.Context()

	info, err := a.storageApp.Info(ctx, pathname)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	if !info.IsDir {
		a.error(w, r, request, model.WrapInvalid(errors.New("regenerate is only available for folder")))
		return
	}

	subset := r.FormValue("subset")

	if len(subset) == 0 {
		a.error(w, r, request, model.WrapInvalid(errors.New("regenerate need a subset")))
		return
	}

	go func(ctx context.Context) {
		var directories []absto.Item

		err := a.storageApp.Walk(ctx, pathname, func(item absto.Item) error {
			if item.IsDir {
				directories = append(directories, item)
			} else {
				a.notify(ctx, provider.NewRestartEvent(item, subset))
			}

			return nil
		})
		if err != nil {
			logger.Error("regenerate of `%s`: %s", pathname, err)
		}

		for _, directory := range directories {
			a.notify(ctx, provider.NewStartEvent(directory))
		}
	}(cntxt.WithoutDeadline(ctx))

	a.rendererApp.Redirect(w, r, "?stats", renderer.NewSuccessMessage("Regeneration of %s in progress...", subset))
}
