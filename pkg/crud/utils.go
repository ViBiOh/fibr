package crud

import (
	"net/http"

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
