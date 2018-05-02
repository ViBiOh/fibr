package crud

import (
	"net/http"
	"os"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func (a *App) getMetadataFileinfo(request *provider.Request, name []byte) (string, os.FileInfo) {
	return provider.GetFileinfoFromRoot(path.Join(a.rootDirectory, provider.MetadataDirectoryName), request, name)
}

func (a *App) getFileinfo(request *provider.Request, name []byte) (string, os.FileInfo) {
	return provider.GetFileinfoFromRoot(a.rootDirectory, request, name)
}

func (a *App) getFormOrPathFilename(r *http.Request, request *provider.Request) (string, os.FileInfo, error) {
	formName := r.FormValue(`name`)
	if formName != `` {
		if formName == `/` {
			return ``, nil, ErrNotAuthorized
		}

		if filename, info := a.getFileinfo(request, []byte(formName)); filename != `` {
			return filename, info, nil
		}
	}

	if request.Path == `/` {
		return ``, nil, ErrNotAuthorized
	}

	filename, info := a.getFileinfo(request, []byte(formName))
	return filename, info, nil
}
