package search

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
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
	tags        []string
	size        int64
	greaterThan bool
}

func (s search) hasTags() bool {
	return len(s.tags) > 0
}

func parseSearch(params url.Values, now time.Time) (output search, err error) {
	if name := strings.TrimSpace(params.Get("name")); len(name) > 0 {
		output.pattern, err = regexp.Compile(name)
		if err != nil {
			return
		}
	}

	if tags := strings.TrimSpace(params.Get("tags")); len(tags) > 0 {
		output.tags = strings.Split(tags, " ")
	}

	output.before, err = parseDate(strings.TrimSpace(params.Get("before")))
	if err != nil {
		return
	}

	output.after, err = parseDate(strings.TrimSpace(params.Get("after")))
	if err != nil {
		return
	}

	if rawSince := params.Get("since"); len(rawSince) != 0 {
		var value int

		value, err = strconv.Atoi(rawSince)
		if err != nil {
			return
		}

		before := computeSince(now, params.Get("sinceUnit"), value)

		if before.After(output.after) {
			output.after = before
		}
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

func (s search) matchTags(metadata provider.Metadata) bool {
	if len(metadata.Tags) == 0 {
		return false
	}

	for _, tag := range s.tags {
		var found bool

		for _, itemTag := range metadata.Tags {
			if itemTag == tag {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}
