package crud

import (
	"encoding/json"
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

// SerializableRegexp is a regexp you can serialize
type SerializableRegexp struct {
	re *regexp.Regexp
}

// MarshalJSON marshals the regexp as a a string
func (sr SerializableRegexp) MarshalJSON() ([]byte, error) {
	if sr.re == nil {
		return nil, nil
	}

	return json.Marshal(sr.re.String())
}

// UnmarshalJSON unmarshal JSOn
func (sr *SerializableRegexp) UnmarshalJSON(b []byte) error {
	var strValue string
	err := json.Unmarshal(b, &strValue)
	if err != nil {
		return fmt.Errorf("unable to unmarshal serializable regexp: %s", err)
	}

	value, err := regexp.Compile(strValue)
	if err != nil {
		return fmt.Errorf("unable to parse serializable regexp: %s", err)
	}

	*sr = SerializableRegexp{
		re: value,
	}
	return nil
}

type search struct {
	Pattern     SerializableRegexp `json:"pattern,omitempty"`
	Before      time.Time          `json:"before,omitempty"`
	After       time.Time          `json:"fter,omitempty"`
	Mimes       []string           `json:"mimes,omitempty"`
	Size        int64              `json:"size,omitempty"`
	GreaterThan bool               `json:"greaterThan,omitempty"`
}

func parseSearch(params url.Values) (output search, err error) {
	if name := strings.TrimSpace(params.Get("name")); len(name) > 0 {
		output.Pattern.re, err = regexp.Compile(name)
		if err != nil {
			return
		}
	}

	output.Before, err = parseDate(strings.TrimSpace(params.Get("before")))
	if err != nil {
		return
	}

	output.After, err = parseDate(strings.TrimSpace(params.Get("after")))
	if err != nil {
		return
	}

	rawSize := strings.TrimSpace(params.Get("size"))
	if len(rawSize) > 0 {
		output.Size, err = strconv.ParseInt(rawSize, 10, 64)
		if err != nil {
			return
		}
	}

	output.Size = computeSize(strings.TrimSpace(params.Get("sizeUnit")), output.Size)
	output.GreaterThan = strings.TrimSpace(params.Get("sizeOrder")) == "gt"
	output.Mimes = computeMimes(params["types"])

	return
}

func (s search) match(item provider.StorageItem) bool {
	if !s.matchSize(item) {
		return false
	}

	if !s.Before.IsZero() && item.Date.After(s.Before) {
		return false
	}

	if !s.After.IsZero() && item.Date.Before(s.After) {
		return false
	}

	if !s.matchMimes(item) {
		return false
	}

	if s.Pattern.re != nil && !s.Pattern.re.MatchString(item.Pathname) {
		return false
	}

	return true
}

func (s search) matchSize(item provider.StorageItem) bool {
	if s.Size == 0 {
		return true
	}

	if (s.Size - item.Size) > 0 == s.GreaterThan {
		return false
	}

	return true
}

func (s search) matchMimes(item provider.StorageItem) bool {
	if len(s.Mimes) == 0 {
		return true
	}

	itemMime := item.Extension()
	for _, mime := range s.Mimes {
		if strings.EqualFold(mime, itemMime) {
			return true
		}
	}

	return false
}

func (a App) searchFiles(criterions search, request provider.Request) (items []provider.RenderItem, err error) {
	err = a.storageApp.Walk(request.Filepath(), func(item provider.StorageItem) error {
		if item.IsDir || !criterions.match(item) {
			return nil
		}

		items = append(items, provider.StorageToRender(item, request))

		return nil
	})

	return
}

func (a App) search(r *http.Request, request provider.Request) (string, int, map[string]interface{}, error) {
	params := r.URL.Query()

	criterions, err := parseSearch(params)
	if err != nil {
		return "", 0, nil, httpModel.WrapInvalid(err)
	}

	items, err := a.searchFiles(criterions, request)
	if err != nil {
		return "", 0, nil, err
	}

	return "search", http.StatusOK, map[string]interface{}{
		"Paths":   getPathParts(request),
		"Files":   items,
		"Search":  params,
		"Request": request,
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
