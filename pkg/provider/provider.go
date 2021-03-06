package provider

import (
	"io"
)

const (
	// MetadataDirectoryName directory when metadata are stored
	MetadataDirectoryName = ".fibr"
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
	// WordExtensions contains extensions of Word
	WordExtensions = map[string]bool{".doc": true, ".docx": true, ".docm": true}
)

// ReadSeekerCloser is a combination of io.Reader, io.Seeker and io.Closer
type ReadSeekerCloser interface {
	Read([]byte) (int, error)
	Seek(int64, int) (int64, error)
	Close() error
}

// Storage describe action on a storage provider
type Storage interface {
	SetIgnoreFn(ignoreFn func(StorageItem) bool)
	Info(pathname string) (StorageItem, error)
	List(pathname string) ([]StorageItem, error)
	WriterTo(pathname string) (io.WriteCloser, error)
	ReaderFrom(pathname string) (ReadSeekerCloser, error)
	Walk(pathname string, walkFn func(StorageItem, error) error) error
	CreateDir(pathname string) error
	Store(pathname string, content io.ReadCloser) error
	Rename(oldName, newName string) error
	Remove(pathname string) error
}
