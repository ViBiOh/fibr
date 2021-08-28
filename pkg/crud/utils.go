package crud

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
)

func getPreviousAndNext(file provider.StorageItem, files []provider.StorageItem) (*provider.StorageItem, *provider.StorageItem) {
	var (
		found    bool
		previous *provider.StorageItem
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
			previous = &files[index]
		}

		if found {
			return previous, &files[index]
		}
	}

	return previous, nil
}

func checkFormName(r *http.Request, formName string) (string, error) {
	name := r.FormValue(formName)
	if name == "" {
		return "", model.WrapInvalid(ErrEmptyName)
	}

	if name == "/" {
		return "", model.WrapForbidden(ErrNotAuthorized)
	}

	return name, nil
}

func checkFolderName(formName string, request provider.Request) (string, error) {
	name := formName
	if name == "" {
		return "", model.WrapInvalid(ErrEmptyFolder)
	}

	if !strings.HasPrefix(name, "/") {
		return "", model.WrapInvalid(ErrAbsoluteFolder)
	}

	if len(request.Share.ID) != 0 {
		shareURIPrefix := fmt.Sprintf("/%s", request.Share.ID)

		if !strings.HasPrefix(name, shareURIPrefix) {
			return "", model.WrapForbidden(ErrNotAuthorized)
		}

		name = strings.TrimPrefix(name, shareURIPrefix)
	}

	return name, nil
}

func getPathParts(uri string) []string {
	cleanURI := strings.Trim(uri, "/")
	if cleanURI == "" {
		return nil
	}

	return strings.Split(cleanURI, "/")
}

func getFormBool(val string) (bool, error) {
	value := val
	if len(value) == 0 {
		return false, nil
	}

	return strconv.ParseBool(value)
}

func getFormDuration(val string) (time.Duration, error) {
	value := val
	if len(value) == 0 {
		return 0, nil
	}

	return time.ParseDuration(fmt.Sprintf("%sh", value))
}
