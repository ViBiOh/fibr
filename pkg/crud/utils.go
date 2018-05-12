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

func getFilepath(r *http.Request, request *provider.Request) (string, error) {
	name := r.FormValue(`name`)
	if name == `` {
		name = request.Path
	}

	if name == `/` {
		return ``, ErrNotAuthorized
	}

	return provider.GetPathname(request, []byte(name)), nil
}

func getFormFilepath(r *http.Request, request *provider.Request, formName string) (string, error) {
	name := r.FormValue(formName)
	if name == `` {
		return ``, ErrEmptyName
	}

	if name == `/` {
		return ``, ErrNotAuthorized
	}

	return provider.GetPathname(request, []byte(name)), nil
}
