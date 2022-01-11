package provider

import (
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
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

	if (first.IsImage() || first.IsVideo()) && (second.IsImage() || second.IsVideo()) {
		return greaterTime(first.Date, second.Date)
	}

	if first.IsImage() || first.IsVideo() {
		return false
	}

	if second.IsImage() || first.IsVideo() {
		return true
	}

	return lowerString(first.Name, second.Name)
}

// RenderItem is a storage item with an id
type RenderItem struct {
	Aggregate
	ID   string
	URL  string
	Path string
	absto.Item
}

// StorageToRender converts Item to RenderItem
func StorageToRender(item absto.Item, request Request) RenderItem {
	return RenderItem{
		ID:   sha.New(item.Name),
		URL:  request.RelativeURL(item),
		Path: request.Path,
		Item: item,
	}
}
