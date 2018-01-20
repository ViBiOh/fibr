package provider

import (
	"net/http"
)

var (
	// ArchiveExtensions contains extensions of Archive
	ArchiveExtensions = map[string]bool{`.zip`: true, `.tar`: true, `.gz`: true, `.rar`: true}
	// AudioExtensions contains extensions of Audio
	AudioExtensions = map[string]bool{`.mp3`: true}
	// CodeExtensions contains extensions of Code
	CodeExtensions = map[string]bool{`.html`: true, `.css`: true, `.js`: true, `.jsx`: true, `.json`: true, `.yml`: true, `.yaml`: true, `.toml`: true, `.md`: true, `.go`: true, `.py`: true, `.java`: true, `.xml`: true}
	// ExcelExtensions contains extensions of Excel
	ExcelExtensions = map[string]bool{`.xls`: true, `.xlsx`: true, `.xlsm`: true}
	// ImageExtensions contains extensions of Image
	ImageExtensions = map[string]bool{`.jpg`: true, `.jpeg`: true, `.png`: true, `.gif`: true}
	// PdfExtensions contains extensions of Pdf
	PdfExtensions = map[string]bool{`.pdf`: true}
	// VideoExtensions contains extensions of Video
	VideoExtensions = map[string]bool{`.mp4`: true, `.mov`: true, `.avi`: true}
	// WordExtensions contains extensions of Word
	WordExtensions = map[string]bool{`.doc`: true, `.docx`: true, `.docm`: true}
)

// RequestConfig stores informations
type RequestConfig struct {
	URL        string
	Root       string
	PathPrefix string
	Path       string
	CanEdit    bool
	CanShare   bool
}

// Message rendered to user
type Message struct {
	Level   string
	Content string
}

// Renderer interface for return rich content to user
type Renderer interface {
	Error(http.ResponseWriter, int, error)
	Sitemap(http.ResponseWriter)
	Directory(http.ResponseWriter, *RequestConfig, map[string]interface{}, string, *Message)
}
