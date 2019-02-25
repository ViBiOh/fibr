package provider

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
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
	CodeExtensions = map[string]bool{`.html`: true, `.css`: true, `.js`: true, `.jsx`: true, `.json`: true, `.yml`: true, `.yaml`: true, `.toml`: true, `.md`: true, `.go`: true, `.py`: true, `.java`: true, `.xml`: true}
	// ExcelExtensions contains extensions of Excel
	ExcelExtensions = map[string]bool{`.xls`: true, `.xlsx`: true, `.xlsm`: true}
	// ImageExtensions contains extensions of Image
	ImageExtensions = map[string]bool{`.jpg`: true, `.jpeg`: true, `.png`: true, `.gif`: true, `.svg`: true, `.tiff`: true}
	// PdfExtensions contains extensions of Pdf
	PdfExtensions = map[string]bool{`.pdf`: true}
	// VideoExtensions contains extensions of Video
	VideoExtensions = map[string]string{`.mp4`: `video/mp4`, `.mov`: `video/quicktime`, `.avi`: `video/x-msvideo`}
	// WordExtensions contains extensions of Word
	WordExtensions = map[string]bool{`.doc`: true, `.docx`: true, `.docm`: true}
)

// Request from user
type Request struct {
	Path     string
	CanEdit  bool
	CanShare bool
	Share    *Share
}

// GetPath of request according to share
func (r Request) GetPath() string {
	var prefix string

	if r.Share != nil {
		prefix = r.Share.Path
	}

	return path.Join(prefix, r.Path)
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
	title := p.Config.Seo.Title

	if p.Request != nil {
		if p.Request.Share != nil {
			title = fmt.Sprintf(`%s - %s`, title, p.Request.Share.RootName)
		} else {
			title = fmt.Sprintf(`%s - %s`, title, p.Config.RootName)
		}

		path := strings.Trim(p.Request.Path, `/`)
		if path != `` {
			title = fmt.Sprintf(`%s - %s`, title, path)
		}
	}

	return title
}

// Description compute title of page
func (p Page) Description() string {
	description := p.Config.Seo.Description

	if p.Request != nil {
		if p.Request.Share != nil {
			description = fmt.Sprintf(`%s - %s`, description, p.Request.Share.RootName)
		} else {
			description = fmt.Sprintf(`%s - %s`, description, p.Config.RootName)
		}

		if p.Request.Path != `` {
			description = fmt.Sprintf(`%s/%s`, description, strings.Trim(p.Request.Path, `/`))
		}
	}

	return description
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
	SVG(http.ResponseWriter, string, string)
	Directory(http.ResponseWriter, *Request, map[string]interface{}, string, *Message)
	File(http.ResponseWriter, *Request, map[string]interface{}, *Message)
}

// StorageItem describe item on a storage provider
type StorageItem struct {
	Pathname string
	Name     string
	IsDir    bool
}

// Extension gives extensions of item
func (s StorageItem) Extension() string {
	return strings.ToLower(path.Ext(s.Name))
}

// Mime gives Mime Type of item
func (s StorageItem) Mime() string {
	extension := s.Extension()
	if mime := mime.TypeByExtension(extension); mime != `` {
		return mime
	}

	if CodeExtensions[extension] {
		return `text/plain`
	}

	if mime, ok := VideoExtensions[extension]; ok {
		return mime
	}

	return ``
}

// IsImage determine if item if an image
func (s StorageItem) IsImage() bool {
	return ImageExtensions[s.Extension()]
}

// IsVideo determine if item if a video
func (s StorageItem) IsVideo() bool {
	return VideoExtensions[s.Extension()] != ``
}

// Storage describe action on a storage provider
type Storage interface {
	Name() string
	Root() string
	Info(pathname string) (*StorageItem, error)
	Open(pathname string) (io.WriteCloser, error)
	Read(pathname string) (io.ReadCloser, error)
	Serve(http.ResponseWriter, *http.Request, string)
	List(pathname string) ([]*StorageItem, error)
	Walk(walkFn func(*StorageItem, error) error) error
	Create(name string) error
	Upload(pathname string, content io.ReadCloser) error
	Rename(oldName, newName string) error
	Remove(pathname string) error
}
