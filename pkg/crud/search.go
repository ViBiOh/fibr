package crud

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
)

const (
	isoDateLayout = "2006-01-02"
)

func (a App) search(r *http.Request, request provider.Request) (string, int, map[string]interface{}, error) {
	var items []provider.RenderItem

	params := r.URL.Query()

	before, err := parseDate(strings.TrimSpace(params.Get("before")))
	if err != nil {
		return "", 0, nil, httpModel.WrapInvalid(err)
	}

	after, err := parseDate(strings.TrimSpace(params.Get("after")))
	if err != nil {
		return "", 0, nil, httpModel.WrapInvalid(err)
	}

	mimes := params["mime"]

	err = a.storageApp.Walk(request.GetFilepath(""), func(item provider.StorageItem) error {
		if item.IsDir {
			return nil
		}

		if !before.IsZero() && item.Date.After(before) {
			return nil
		}

		if !after.IsZero() && item.Date.Before(after) {
			return nil
		}

		if !matchMimes(item, mimes) {
			return nil
		}

		items = append(items, provider.RenderItem{
			ID:          sha.New(item.Name),
			URI:         request.URL(""),
			StorageItem: item,
		})

		return nil
	})

	if err != nil {
		return "", 0, nil, err
	}

	return "search", http.StatusOK, map[string]interface{}{
		"Paths": getPathParts(request.URL("")),
		"Files": items,

		"Request": request,
	}, nil
}

func matchMimes(item provider.StorageItem, mimes []string) bool {
	if len(mimes) == 0 {
		return true
	}

	itemMime := item.Mime()
	for _, mime := range mimes {
		if strings.EqualFold(mime, itemMime) {
			return true
		}
	}

	return false
}

func parseDate(raw string) (time.Time, error) {
	if len(raw) == 0 {
		return time.Time{}, nil
	}

	value, err := time.Parse(isoDateLayout, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to parse date: %s", err)
	}

	return value, nil
}
