package provider

import (
	"mime"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/sha"
)

// StorageItem describe item on a storage provider
type StorageItem struct {
	Date     time.Time `json:"date"`
	Name     string    `json:"name"`
	Pathname string    `json:"pathname"`
	IsDir    bool      `json:"isDir"`
	Size     int64     `json:"size"`
}

// Extension gives extensions of item
func (s StorageItem) Extension() string {
	return strings.ToLower(path.Ext(s.Name))
}

// Mime gives Mime Type of item
func (s StorageItem) Mime() string {
	extension := s.Extension()
	if mimeType := mime.TypeByExtension(extension); mimeType != "" {
		return mimeType
	}

	if CodeExtensions[extension] {
		return "text/plain; charset=utf-8"
	}

	return ""
}

// IsPdf determine if item if a pdf
func (s StorageItem) IsPdf() bool {
	return PdfExtensions[s.Extension()]
}

// IsImage determine if item if an image
func (s StorageItem) IsImage() bool {
	return ImageExtensions[s.Extension()]
}

// IsVideo determine if item if a video
func (s StorageItem) IsVideo() bool {
	return VideoExtensions[s.Extension()] != ""
}

// Dir return the nearest directory (self of parent)
func (s StorageItem) Dir() string {
	if s.IsDir {
		return s.Pathname
	}

	return filepath.Dir(s.Pathname)
}

func lowerString(first, second string) bool {
	return strings.Compare(strings.ToLower(first), strings.ToLower(second)) < 0
}

func greaterTime(first, second time.Time) bool {
	return first.After(second)
}

// ByHybridSort implements Sorter by type, name then modification time
type ByHybridSort []StorageItem

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
	StorageItem
}

// StorageToRender converts StorageItem to RenderItem
func StorageToRender(item StorageItem, request Request) RenderItem {
	return RenderItem{
		ID:          sha.New(item.Name),
		URL:         request.RelativeURL(item),
		Path:        request.Path,
		StorageItem: item,
	}
}
