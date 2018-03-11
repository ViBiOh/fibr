package crud

import (
	"net/http"
	"os"

	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/fibr/utils"
)

func (a *App) getFormOrPathFilename(r *http.Request, config *provider.RequestConfig) (string, os.FileInfo, error) {
	formName := r.FormValue(`name`)
	if formName != `` {
		if formName == `/` {
			return ``, nil, ErrNotAuthorized
		}

		if filename, info := utils.GetPathInfo(a.rootDirectory, config.Root, config.Path, formName); filename != `` {
			return filename, info, nil
		}
	}

	if config.Path == `/` {
		return ``, nil, ErrNotAuthorized
	}

	filename, info := utils.GetPathInfo(a.rootDirectory, config.Root, config.Path, formName)
	return filename, info, nil
}
