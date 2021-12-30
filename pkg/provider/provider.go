package provider

import (
	"io"
	"time"
)

const (
	// MetadataDirectoryName directory where metadata are stored
	MetadataDirectoryName = "/.fibr"
)

var (
	// ArchiveExtensions contains extensions of Archive
	ArchiveExtensions = map[string]bool{".zip": true, ".tar": true, ".gz": true, ".rar": true}
	// AudioExtensions contains extensions of Audio
	AudioExtensions = map[string]bool{".mp3": true}
	// CodeExtensions contains extensions of Code
	CodeExtensions = map[string]bool{".html": true, ".css": true, ".js": true, ".jsx": true, ".json": true, ".yml": true, ".yaml": true, ".toml": true, ".md": true, ".go": true, ".py": true, ".java": true, ".xml": true}
	// ExcelExtensions contains extensions of Excel
	ExcelExtensions = map[string]bool{".xls": true, ".xlsx": true, ".xlsm": true}
	// ImageExtensions contains extensions of Image
	ImageExtensions = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".svg": true, ".tiff": true}
	// PdfExtensions contains extensions of Pdf
	PdfExtensions = map[string]bool{".pdf": true}
	// VideoExtensions contains extensions of Video
	VideoExtensions = map[string]string{".mp4": "video/mp4", ".mov": "video/quicktime", ".avi": "video/x-msvideo"}
	// StreamExtensions contains extensions of streamable content
	StreamExtensions = map[string]bool{".ts": true}
	// WordExtensions contains extensions of Word
	WordExtensions = map[string]bool{".doc": true, ".docx": true, ".docm": true}
)

// Storage describe action on a storage provider
//go:generate mockgen -destination ../mocks/storage.go -mock_names Storage=Storage -package mocks github.com/ViBiOh/fibr/pkg/provider Storage
type Storage interface {
	WithIgnoreFn(ignoreFn func(StorageItem) bool) Storage
	Info(pathname string) (StorageItem, error)
	List(pathname string) ([]StorageItem, error)
	WriterTo(pathname string) (io.WriteCloser, error)
	ReaderFrom(pathname string) (io.ReadSeekCloser, error)
	Walk(pathname string, walkFn func(StorageItem) error) error
	CreateDir(pathname string) error
	Rename(oldName, newName string) error
	Remove(pathname string) error
	UpdateDate(pathname string, date time.Time) error
}
