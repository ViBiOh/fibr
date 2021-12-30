package crud

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
)

const (
	isoDateLayout = "2006-01-02"
)

func (a App) search(r *http.Request, request provider.Request) (string, int, map[string]interface{}, error) {
	var items []provider.RenderItem

	params := r.URL.Query()

	var pattern *regexp.Regexp
	name := strings.TrimSpace(params.Get("name"))
	if len(name) > 0 {
		var err error
		pattern, err = regexp.Compile(name)
		if err != nil {
			return "", 0, nil, httpModel.WrapInvalid(err)
		}
	}

	before, err := parseDate(strings.TrimSpace(params.Get("before")))
	if err != nil {
		return "", 0, nil, httpModel.WrapInvalid(err)
	}

	after, err := parseDate(strings.TrimSpace(params.Get("after")))
	if err != nil {
		return "", 0, nil, httpModel.WrapInvalid(err)
	}

	mimes := computeMimes(params["types"])

	err = a.storageApp.Walk(request.Filepath(), func(item provider.StorageItem) error {
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

		if pattern != nil && !pattern.MatchString(item.Pathname) {
			return nil
		}

		items = append(items, provider.StorageToRender(item, request))

		return nil
	})

	if err != nil {
		return "", 0, nil, err
	}

	return "search", http.StatusOK, map[string]interface{}{
		"Paths":  getPathParts(request.Path),
		"Files":  items,
		"Search": params,

		"Request": request,
	}, nil
}

func computeMimes(aliases []string) []string {
	var output []string

	for _, alias := range aliases {
		switch alias {
		case "archive":
			output = append(output, getKeysOfMapBool(provider.ArchiveExtensions)...)
		case "audio":
			output = append(output, getKeysOfMapBool(provider.AudioExtensions)...)
		case "code":
			output = append(output, getKeysOfMapBool(provider.CodeExtensions)...)
		case "excel":
			output = append(output, getKeysOfMapBool(provider.ExcelExtensions)...)
		case "image":
			output = append(output, getKeysOfMapBool(provider.ImageExtensions)...)
		case "pdf":
			output = append(output, getKeysOfMapBool(provider.PdfExtensions)...)
		case "video":
			output = append(output, getKeysOfMapString(provider.VideoExtensions)...)
		case "stream":
			output = append(output, getKeysOfMapBool(provider.StreamExtensions)...)
		case "word":
			output = append(output, getKeysOfMapBool(provider.WordExtensions)...)
		}
	}

	return output
}

func getKeysOfMapBool(input map[string]bool) []string {
	output := make([]string, len(input))
	var i int64

	for key := range input {
		output[i] = key
		i++
	}

	return output
}

func getKeysOfMapString(input map[string]string) []string {
	output := make([]string, len(input))
	var i int64

	for key := range input {
		output[i] = key
		i++
	}

	return output
}

func matchMimes(item provider.StorageItem, mimes []string) bool {
	if len(mimes) == 0 {
		return true
	}

	itemMime := item.Extension()
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
