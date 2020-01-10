package crud

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func getPreviousAndNext(file provider.StorageItem, files []provider.StorageItem) (*provider.StorageItem, *provider.StorageItem) {
	var (
		found    bool
		previous provider.StorageItem
	)

	for index, neighbor := range files {
		if neighbor.IsDir != file.IsDir {
			continue
		}

		if neighbor.Name == file.Name {
			found = true
			continue
		}

		if !found {
			previous = files[index]
		}

		if found {
			return &previous, &files[index]
		}
	}

	return &previous, nil
}

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

func getPathParts(uri string) []string {
	cleanURI := strings.TrimSpace(strings.Trim(uri, "/"))
	if cleanURI == "" {
		return nil
	}

	return strings.Split(cleanURI, "/")
}
