package crud

import (
	"errors"
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// Regenerate regenerate start of the folder
func (a App) Regenerate(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	pathname := request.Filepath()

	info, err := a.storageApp.Info(pathname)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	if !info.IsDir {
		a.error(w, r, request, model.WrapInvalid(errors.New("regenerate is only available for folder")))
	}

	if subset := r.FormValue("subset"); len(subset) != 0 {
		go func() {
			err := a.storageApp.Walk(pathname, func(item absto.Item) error {
				a.notify(provider.NewRestartEvent(item, subset))
				return nil
			})
			if err != nil {
				logger.Error("error during regenerate of `%s`: %s", pathname, err)
			}
		}()
	}

	a.rendererApp.Redirect(w, r, "?stats", renderer.NewSuccessMessage("Regeneration in progress..."))
}
