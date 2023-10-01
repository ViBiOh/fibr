package provider

import (
	"path/filepath"

	absto "github.com/ViBiOh/absto/pkg/model"
)

const (
	MetadataDirectoryName = "/.fibr"
	MaxConcurrency        = 6
	MaxClientSideCaching  = 2000
)

var (
	ArchiveExtensions = map[string]bool{".zip": true, ".tar": true, ".gz": true, ".rar": true}
	AudioExtensions   = map[string]bool{".mp3": true}
	CodeExtensions    = map[string]bool{".html": true, ".css": true, ".js": true, ".jsx": true, ".json": true, ".yml": true, ".yaml": true, ".toml": true, ".md": true, ".go": true, ".py": true, ".java": true, ".xml": true}
	ExcelExtensions   = map[string]bool{".xls": true, ".xlsx": true, ".xlsm": true}
	ImageExtensions   = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".svg": true, ".tiff": true, ".webp": true, ".heic": true}
	PdfExtensions     = map[string]bool{".pdf": true}
	VideoExtensions   = map[string]string{".mp4": "video/mp4", ".mov": "video/mp4", ".avi": "video/x-msvideo", ".ogg": "video/ogg", ".mkv": "video/x-matroska"}
	StreamExtensions  = map[string]bool{".ts": true}
	WordExtensions    = map[string]bool{".doc": true, ".docx": true, ".docm": true}

	// ThumbnailExtensions contains extensions of file eligible to thumbnail
	ThumbnailExtensions = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".tiff": true, ".webp": true, ".pdf": true, ".mp4": true, ".mov": true, ".avi": true, ".ogg": true, ".mkv": true}
)

func MetadataDirectory(item absto.Item) string {
	pathname := item.Pathname
	if !item.IsDir() {
		pathname = filepath.Dir(pathname)
	}

	return Dirname(MetadataDirectoryName + pathname)
}
