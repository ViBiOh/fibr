package provider

import (
	"mime"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/exas/pkg/model"
)

func lowerString(first, second string) bool {
	return strings.Compare(strings.ToLower(first), strings.ToLower(second)) < 0
}

func greaterTime(first, second time.Time) bool {
	return first.After(second)
}

// ByHybridSort implements Sorter by type then modification time
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

type ByID []absto.Item

func (a ByID) Len() int      { return len(a) }
func (a ByID) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByID) Less(i, j int) bool {
	return a[i].ID < a[j].ID
}

type RenderItem struct {
	Aggregate
	URL  string
	Path string
	absto.Item
	HasThumbnail bool
	IsCover      bool
}

func (r RenderItem) IsZero() bool {
	return r.Item.IsZero()
}

func (r RenderItem) IsImage() bool {
	_, ok := ImageExtensions[r.Extension]
	return ok
}

func (r RenderItem) IsVideo() bool {
	_, ok := VideoExtensions[r.Extension]
	return ok
}

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

func StorageToRender(item absto.Item, request Request) RenderItem {
	return RenderItem{
		URL:  request.RelativeURL(item),
		Path: request.Path,
		Item: item,
	}
}

type StoryItem struct {
	Exif model.Exif
	RenderItem
}

func StorageToStory(item absto.Item, request Request, exif model.Exif) StoryItem {
	return StoryItem{
		RenderItem: StorageToRender(item, request),
		Exif:       exif,
	}
}
