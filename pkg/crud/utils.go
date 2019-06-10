package crud

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func checkFormName(r *http.Request, formName string) (string, error) {
	name := strings.TrimSpace(r.FormValue(formName))
	if name == "" {
		return "", ErrEmptyName
	}

	if name == "/" {
		return "", ErrNotAuthorized
	}

	return name, nil
}

func getPathParts(request *provider.Request) []string {
	cleanURI := strings.TrimSpace(strings.Trim(request.GetURI(""), "/"))
	if cleanURI == "" {
		return nil
	}

	return strings.Split(cleanURI, "/")
}
