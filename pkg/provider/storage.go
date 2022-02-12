package provider

import (
	"mime"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
)

func lowerString(first, second string) bool {
	return strings.Compare(strings.ToLower(first), strings.ToLower(second)) < 0
}

func greaterTime(first, second time.Time) bool {
	return first.After(second)
}

// ByHybridSort implements Sorter by type, name then modification time
type ByHybridSort []absto.Item

func (a ByHybridSort) Len() int {
	return len(a)
}

func (a ByHybridSort) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByHybridSort) Less(i, j int) bool {
	first := a[i]
	second := a[j]

	if first.IsDir && second.IsDir {
		return lowerString(first.Name, second.Name)
	}

	if first.IsDir {
		return true
	}

	if second.IsDir {
		return false
	}

	return greaterTime(first.Date, second.Date)
}

// RenderItem is a storage item with an id
type RenderItem struct {
	Aggregate
	URL  string
	Path string
	absto.Item
	HasThumbnail bool
}

// IsImage check if item is an image
func (r RenderItem) IsImage() bool {
	_, ok := ImageExtensions[r.Extension]
	return ok
}

// IsVideo check if item is an image
func (r RenderItem) IsVideo() bool {
	_, ok := VideoExtensions[r.Extension]
	return ok
}

// Mime gives Mime Type of item
func (r RenderItem) Mime() string {
	if mimeType, ok := VideoExtensions[r.Extension]; ok {
		return mimeType
	}

	if mimeType := mime.TypeByExtension(r.Extension); mimeType != "" {
		return mimeType
	}

	if CodeExtensions[r.Extension] {
		return "text/plain; charset=utf-8"
	}

	return ""
}

// StorageToRender converts Item to RenderItem
func StorageToRender(item absto.Item, request Request) RenderItem {
	return RenderItem{
		URL:  request.RelativeURL(item),
		Path: request.Path,
		Item: item,
	}
}
