package crud

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
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

type search struct {
	pattern     *regexp.Regexp
	before      time.Time
	after       time.Time
	mimes       []string
	size        int64
	greaterThan bool
}

func parseSearch(params url.Values) (output search, err error) {
	if name := strings.TrimSpace(params.Get("name")); len(name) > 0 {
		output.pattern, err = regexp.Compile(name)
		if err != nil {
			return
		}
	}

	output.before, err = parseDate(strings.TrimSpace(params.Get("before")))
	if err != nil {
		return
	}

	output.after, err = parseDate(strings.TrimSpace(params.Get("after")))
	if err != nil {
		return
	}

	rawSize := strings.TrimSpace(params.Get("size"))
	if len(rawSize) > 0 {
		output.size, err = strconv.ParseInt(rawSize, 10, 64)
		if err != nil {
			return
		}
	}

	output.size = computeSize(strings.TrimSpace(params.Get("sizeUnit")), output.size)
	output.greaterThan = strings.TrimSpace(params.Get("sizeOrder")) == "gt"
	output.mimes = computeMimes(params["types"])

	return
}

func (s search) match(item provider.StorageItem) bool {
	if !s.matchSize(item) {
		return false
	}

	if !s.before.IsZero() && item.Date.After(s.before) {
		return false
	}

	if !s.after.IsZero() && item.Date.Before(s.after) {
		return false
	}

	if !s.matchMimes(item) {
		return false
	}

	if s.pattern != nil && !s.pattern.MatchString(item.Pathname) {
		return false
	}

	return true
}

func (s search) matchSize(item provider.StorageItem) bool {
	if s.size == 0 {
		return true
	}

	if (s.size - item.Size) > 0 == s.greaterThan {
		return false
	}

	return true
}

func (s search) matchMimes(item provider.StorageItem) bool {
	if len(s.mimes) == 0 {
		return true
	}

	itemMime := item.Extension()
	for _, mime := range s.mimes {
		if strings.EqualFold(mime, itemMime) {
			return true
		}
	}

	return false
}

func (a App) searchFiles(r *http.Request, request provider.Request) (items []provider.StorageItem, err error) {
	params := r.URL.Query()

	criterions, err := parseSearch(params)
	if err != nil {
		return nil, httpModel.WrapInvalid(err)
	}

	err = a.storageApp.Walk(request.Filepath(), func(item provider.StorageItem) error {
		if item.IsDir || !criterions.match(item) {
			return nil
		}

		items = append(items, item)

		return nil
	})

	sort.Sort(provider.ByHybridSort(items))

	return
}

func (a App) search(r *http.Request, request provider.Request, files []provider.StorageItem) (string, int, map[string]interface{}, error) {
	items := make([]provider.RenderItem, len(files))
	var hasMap bool

	for i, item := range files {
		items[i] = provider.StorageToRender(item, request)

		if !hasMap {
			if exif, err := a.exifApp.GetExifFor(item); err == nil && exif.Geocode.Longitude != 0 && exif.Geocode.Latitude != 0 {
				hasMap = true
			}
		}

	}

	return "search", http.StatusOK, map[string]interface{}{
		"Paths":   getPathParts(request),
		"Files":   items,
		"Search":  r.URL.Query(),
		"Request": request,
		"HasMap":  hasMap,
	}, nil
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
