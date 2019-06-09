package provider

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strings"
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

	protocolRegex = regexp.MustCompile("^(https?):/")
)

// Request from user
type Request struct {
	Path     string
	CanEdit  bool
	CanShare bool
	Share    *Share
}

// GetPath of request
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
	Content map[string]interface{}

	PublicURL   string
	Title       string
	Description string
}

// PageBuilder for interactively create page
type PageBuilder struct {
	config  *Config
	request *Request
	message *Message
	error   *Error
	layout  string
	content map[string]interface{}
}

// Config set Config for page
func (p *PageBuilder) Config(config *Config) *PageBuilder {
	p.config = config

	return p
}

// Request set Request for page
func (p *PageBuilder) Request(request *Request) *PageBuilder {
	p.request = request

	return p
}

// Message set Message for page
func (p *PageBuilder) Message(message *Message) *PageBuilder {
	p.message = message

	return p
}

// Error set Error for page
func (p *PageBuilder) Error(error *Error) *PageBuilder {
	p.error = error

	return p
}

// Layout set Layout for page
func (p *PageBuilder) Layout(layout string) *PageBuilder {
	p.layout = layout

	return p
}

// Content set content for page
func (p *PageBuilder) Content(content map[string]interface{}) *PageBuilder {
	p.content = content

	return p
}

// Build Page Object
func (p *PageBuilder) Build() Page {
	layout := p.layout
	var publicURL, title, description string

	if p.config != nil && p.request != nil {
		publicURL = computePublicURL(p.config, p.request)
		title = computeTitle(p.config, p.request)
		description = computeDescription(p.config, p.request)
	}

	if p.layout == "" {
		layout = "grid"
	}

	return Page{
		Config:  p.config,
		Request: p.request,
		Message: p.message,
		Error:   p.error,
		Layout:  layout,
		Content: p.content,

		PublicURL:   publicURL,
		Title:       title,
		Description: description,
	}
}

func computePublicURL(config *Config, request *Request) string {
	parts := []string{config.PublicURL}

	if request != nil {
		if request.Share != nil {
			parts = append(parts, request.Share.ID)
		}

		parts = append(parts, request.Path)
	}

	return protocolRegex.ReplaceAllString(path.Join(parts...), "$1://")
}

func computeTitle(config *Config, request *Request) string {
	title := config.Seo.Title

	if request != nil {
		if request.Share != nil {
			title = fmt.Sprintf("%s - %s", title, request.Share.RootName)
		} else {
			title = fmt.Sprintf("%s - %s", title, config.RootName)
		}

		path := strings.Trim(request.Path, "/")
		if path != "" {
			title = fmt.Sprintf("%s - %s", title, path)
		}
	}

	return title
}

func computeDescription(config *Config, request *Request) string {
	description := config.Seo.Description

	if request != nil {
		if request.Share != nil {
			description = fmt.Sprintf("%s - %s", description, request.Share.RootName)
		} else {
			description = fmt.Sprintf("%s - %s", description, config.RootName)
		}

		if request.Path != "" {
			description = fmt.Sprintf("%s/%s", description, strings.Trim(request.Path, "/"))
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
	Status  int
	Content string
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
	if mime := mime.TypeByExtension(extension); mime != "" {
		return mime
	}

	if CodeExtensions[extension] {
		return "text/plain"
	}

	if mime, ok := VideoExtensions[extension]; ok {
		return mime
	}

	return ""
}

// IsImage determine if item if an image
func (s StorageItem) IsImage() bool {
	return ImageExtensions[s.Extension()]
}

// IsVideo determine if item if a video
func (s StorageItem) IsVideo() bool {
	return VideoExtensions[s.Extension()] != ""
}

// Storage describe action on a storage provider
type Storage interface {
	Name() string
	Root() string
	Info(pathname string) (*StorageItem, error)
	WriterTo(pathname string) (io.WriteCloser, error)
	ReaderFrom(pathname string) (io.ReadCloser, error)
	Serve(http.ResponseWriter, *http.Request, string)
	List(pathname string) ([]*StorageItem, error)
	Walk(walkFn func(*StorageItem, error) error) error
	CreateDir(name string) error
	Store(pathname string, content io.ReadCloser) error
	Rename(oldName, newName string) error
	Remove(pathname string) error
}
