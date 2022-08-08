package provider

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"path"
	"path/filepath"

	absto "github.com/ViBiOh/absto/pkg/model"
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
	ImageExtensions = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".svg": true, ".tiff": true, ".webp": true}
	// PdfExtensions contains extensions of Pdf
	PdfExtensions = map[string]bool{".pdf": true}
	// VideoExtensions contains extensions of Video
	VideoExtensions = map[string]string{".mp4": "video/mp4", ".mov": "video/mp4", ".avi": "video/x-msvideo", ".ogg": "video/ogg", ".mkv": "video/x-matroska"}
	// StreamExtensions contains extensions of streamable content
	StreamExtensions = map[string]bool{".ts": true}
	// WordExtensions contains extensions of Word
	WordExtensions = map[string]bool{".doc": true, ".docx": true, ".docm": true}

	// ThumbnailExtensions contains extensions of file eligible to thumbnail
	ThumbnailExtensions = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".tiff": true, ".webp": true, ".pdf": true, ".mp4": true, ".mov": true, ".avi": true, ".ogg": true, ".mkv": true}
)

// MetadataDirectory computes metadata directory for given item
func MetadataDirectory(item absto.Item) string {
	pathname := item.Pathname
	if !item.IsDir {
		pathname = filepath.Dir(pathname)
	}

	return Dirname(MetadataDirectoryName + pathname)
}

func itemPreviousID(item absto.Item) string {
	hasher := sha1.New()
	_, _ = hasher.Write([]byte(item.Pathname))
	return hex.EncodeToString(hasher.Sum(nil))
}

func RenamePreviousID(ctx context.Context, storageApp absto.Storage, item absto.Item, suffix string, renameFunc func(context.Context, absto.Item, absto.Item) error) error {
	if _, err := storageApp.Info(ctx, path.Join(MetadataDirectory(item), itemPreviousID(item)+suffix)); err != nil {
		if absto.IsNotExist(err) {
			return nil
		}
		return err
	}

	previous := item
	previous.ID = itemPreviousID(previous)

	return renameFunc(ctx, previous, item)
}
