package crud

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func getFilepath(r *http.Request, request *provider.Request) (string, error) {
	name := r.FormValue("name")
	if name == "" {
		name = request.Path
	}

	if name == "/" {
		return "", ErrNotAuthorized
	}

	return request.GetFilepath(name), nil
}

func getFormFilepath(r *http.Request, request *provider.Request, formName string) (string, error) {
	name := r.FormValue(formName)
	if name == "" {
		return "", ErrEmptyName
	}

	if name == "/" {
		return "", ErrNotAuthorized
	}

	return request.GetFilepath(name), nil
}

func getPathParts(request *provider.Request) []string {
	cleanURI := strings.TrimSpace(strings.Trim(request.GetURI(""), "/"))
	if cleanURI == "" {
		return nil
	}

	return strings.Split(cleanURI, "/")
}
