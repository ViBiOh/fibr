package provider

import (
	"fmt"
	"net/http"
	"strings"
)

// MetadataDirectoryName directory when metadata are stored
const MetadataDirectoryName = `.fibr`

var (
	// ArchiveExtensions contains extensions of Archive
	ArchiveExtensions = map[string]bool{`.zip`: true, `.tar`: true, `.gz`: true, `.rar`: true}
	// AudioExtensions contains extensions of Audio
	AudioExtensions = map[string]bool{`.mp3`: true}
	// CodeExtensions contains extensions of Code
	CodeExtensions = map[string]bool{`.html`: true, `.css`: true, `.js`: true, `.jsx`: true, `.json`: true, `.yml`: true, `.yaml`: true, `.toml`: true, `.md`: true, `.go`: true, `.gohtml`: true, `.py`: true, `.java`: true, `.xml`: true}
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

// Request from user
type Request struct {
	Path     string
	CanEdit  bool
	CanShare bool
	IsDebug  bool
	Share    *Share
}

// Share stores informations about shared paths
type Share struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	RootName string `json:"rootName"`
	Edit     bool   `json:"edit"`
	Password string `json:"password"`
}

// Page renderer to user
type Page struct {
	Config  *Config
	Request *Request
	Message *Message
	Error   *Error
	Layout  string
	Path    string
	Content map[string]interface{}
}

// PublicURL compute public URL
func (p Page) PublicURL() string {
	url := p.Config.PublicURL

	if p.Request != nil {
		if p.Request.Share != nil {
			url = fmt.Sprintf(`%s/%s`, url, p.Request.Share.ID)
		}

		url = fmt.Sprintf(`%s%s`, url, p.Request.Path)
	}

	return url
}

// Title compute title of page
func (p Page) Title() string {
	title := fmt.Sprintf(`%s - %s`, p.Config.Seo.Title, p.Config.RootName)

	if p.Request != nil {
		title = fmt.Sprintf(`%s - %s`, title, strings.Trim(p.Request.Path, `/`))
	}

	return title
}

// Config data
type Config struct {
	RootName  string
	PublicURL string
	Version   string
	Seo       *Seo
}

// Seo data
type Seo struct {
	Title       string
	Description string
	Img         string
	ImgHeight   uint
	ImgWidth    uint
}

// Message rendered to user
type Message struct {
	Level   string
	Content string
}

// Error rendered to user
type Error struct {
	Status int
}

// Renderer interface for return rich content to user
type Renderer interface {
	Error(http.ResponseWriter, int, error)
	Sitemap(http.ResponseWriter)
	Directory(http.ResponseWriter, *Request, map[string]interface{}, string, *Message)
}
