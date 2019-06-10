package crud

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func checkFormName(r *http.Request, formName string) (string, *provider.Error) {
	name := strings.TrimSpace(r.FormValue(formName))
	if name == "" {
		return "", provider.NewError(http.StatusBadRequest, ErrEmptyName)
	}

	if name == "/" {
		return "", provider.NewError(http.StatusForbidden, ErrNotAuthorized)
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
