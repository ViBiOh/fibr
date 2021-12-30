package crud

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
)

const (
	isoDateLayout = "2006-01-02"
	kilobytes     = 1 << 10
	megabytes     = 1 << 20
	gigabytes     = 1 << 30
)

func (a App) search(r *http.Request, request provider.Request) (string, int, map[string]interface{}, error) {
	var items []provider.RenderItem
	var err error

	params := r.URL.Query()

	pattern, before, after, size, greaterThan, err := parseSearch(params)
	if err != nil {
		return "", 0, nil, httpModel.WrapInvalid(err)
	}

	mimes := computeMimes(params["types"])

	err = a.storageApp.Walk(request.Filepath(), func(item provider.StorageItem) error {
		if item.IsDir || !match(item, size, greaterThan, before, after, mimes, pattern) {
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

func parseSearch(params url.Values) (pattern *regexp.Regexp, before, after time.Time, size int64, greaterThan bool, err error) {
	if name := strings.TrimSpace(params.Get("name")); len(name) > 0 {
		pattern, err = regexp.Compile(name)
		if err != nil {
			return
		}
	}

	before, err = parseDate(strings.TrimSpace(params.Get("before")))
	if err != nil {
		return
	}

	after, err = parseDate(strings.TrimSpace(params.Get("after")))
	if err != nil {
		return
	}

	rawSize := strings.TrimSpace(params.Get("size"))
	if len(rawSize) > 0 {
		size, err = strconv.ParseInt(rawSize, 10, 64)
		if err != nil {
			return
		}
	}

	size = computeSize(strings.TrimSpace(params.Get("sizeUnit")), size)
	greaterThan = strings.TrimSpace(params.Get("sizeOrder")) == "gt"

	return
}

func computeSize(unit string, size int64) int64 {
	switch unit {
	case "kb":
		return kilobytes * size
	case "mb":
		return megabytes * size
	case "gb":
		return gigabytes * size
	default:
		return size
	}
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

func match(item provider.StorageItem, size int64, greaterThan bool, before, after time.Time, mimes []string, pattern *regexp.Regexp) bool {
	if !matchSize(item, size, greaterThan) {
		return false
	}

	if !before.IsZero() && item.Date.After(before) {
		return false
	}

	if !after.IsZero() && item.Date.Before(after) {
		return false
	}

	if !matchMimes(item, mimes) {
		return false
	}

	if pattern != nil && !pattern.MatchString(item.Pathname) {
		return false
	}

	return true
}

func matchSize(item provider.StorageItem, size int64, greaterThan bool) bool {
	if size == 0 {
		return true
	}

	if (size - item.Size) > 0 == greaterThan {
		return false
	}

	return true
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
