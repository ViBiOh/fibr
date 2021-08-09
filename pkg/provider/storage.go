package provider

import (
	"mime"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// StorageItem describe item on a storage provider
type StorageItem struct {
	Info     interface{}
	Date     time.Time
	Pathname string
	Name     string
	Size     int64

	IsDir bool
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

// RenderItem is a storage item with an id
type RenderItem struct {
	ID  string
	URI string
	StorageItem
	Aggregate
}
