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
		if neighbor.IsDir {
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
	if len(name) == 0 {
		return "", model.WrapInvalid(ErrEmptyName)
	}

	if name == "/" {
		return "", model.WrapForbidden(ErrNotAuthorized)
	}

	return name, nil
}

func checkFolderName(name string, request provider.Request) (string, error) {
	if len(name) == 0 {
		return "", model.WrapInvalid(ErrEmptyFolder)
	}

	if !strings.HasPrefix(name, "/") {
		return "", model.WrapInvalid(ErrAbsoluteFolder)
	}

	return name, nil
}

func getPathParts(request provider.Request) []string {
	cleanURI := strings.Trim(request.AbsoluteURL(""), "/")
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

func errorReturn(request provider.Request, err error) (string, int, map[string]interface{}, error) {
	return "", 0, map[string]interface{}{"Request": request}, err
}
