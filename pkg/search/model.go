package search

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
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

func (s search) match(item absto.Item) bool {
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

func (s search) matchSize(item absto.Item) bool {
	if s.size == 0 {
		return true
	}

	if (s.size - item.Size) > 0 == s.greaterThan {
		return false
	}

	return true
}

func (s search) matchMimes(item absto.Item) bool {
	if len(s.mimes) == 0 {
		return true
	}

	for _, mime := range s.mimes {
		if strings.EqualFold(mime, item.Extension) {
			return true
		}
	}

	return false
}
