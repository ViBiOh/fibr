package crud

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func getFilepath(r *http.Request, request *provider.Request) (string, error) {
	name := r.FormValue(`name`)
	if name == `` {
		name = request.Path
	}

	if name == `/` {
		return ``, ErrNotAuthorized
	}

	return provider.GetPathname(request, name), nil
}

func getFormFilepath(r *http.Request, request *provider.Request, formName string) (string, error) {
	name := r.FormValue(formName)
	if name == `` {
		return ``, ErrEmptyName
	}

	if name == `/` {
		return ``, ErrNotAuthorized
	}

	return provider.GetPathname(request, name), nil
}

func getPathParts(request *provider.Request) []string {
	paths := strings.Split(strings.Trim(request.Path, `/`), `/`)
	if len(paths) == 1 && paths[0] == `` {
		paths = nil
	}

	return paths
}
