package provider

import (
	"encoding/base64"
	"errors"
	"mime"
	"path"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Preferences holds preferences of the user
type Preferences struct {
	ListLayoutPath []string
}

// Request from user
type Request struct {
	Path        string
	CanEdit     bool
	CanShare    bool
	Display     string
	Preferences Preferences
	Share       *Share
}

// GetFilepath of request
func (r Request) GetFilepath(name string) string {
	parts := make([]string, 0)

	if r.Share != nil {
		parts = append(parts, r.Share.Path)
	}

	parts = append(parts, r.Path)

	if len(name) > 0 {
		parts = append(parts, name)
	}

	return path.Join(parts...)
}

// GetURI of request
func (r Request) GetURI(name string) string {
	parts := make([]string, 0)

	if r.Share != nil {
		parts = append(parts, "/", r.Share.ID)
	}

	parts = append(parts, r.Path)

	if len(name) > 0 {
		parts = append(parts, name)
	}

	return path.Join(parts...)
}

// Layout returns layout of given name based on preferences
func (r Request) Layout(name string) string {
	return r.LayoutPath(strings.Trim(r.GetURI(name), "/"))
}

// LayoutPath returns layout of given path based on preferences
func (r Request) LayoutPath(path string) string {
	if FindIndex(r.Preferences.ListLayoutPath, path) != -1 {
		return "list"
	}
	return "grid"
}

// Share stores informations about shared paths
type Share struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	RootName string `json:"rootName"`
	Edit     bool   `json:"edit"`
	Password string `json:"password"`
	File     bool   `json:"file"`
}

// CheckPassword verifies that request has correct password for share
func (s Share) CheckPassword(authorizationHeader string) error {
	if s.Password == "" {
		return nil
	}

	if authorizationHeader == "" {
		return errors.New("empty authorization header")
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authorizationHeader, "Basic "))
	if err != nil {
		return err
	}

	dataStr := string(data)

	sepIndex := strings.Index(dataStr, ":")
	if sepIndex < 0 {
		return errors.New("invalid format for basic auth")
	}

	password := dataStr[sepIndex+1:]
	if err := bcrypt.CompareHashAndPassword([]byte(s.Password), []byte(password)); err != nil {
		return errors.New("invalid credentials")
	}

	return nil
}

// Config data
type Config struct {
	PublicURL string
	Version   string
	Seo       Seo
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
	Content interface{}
}

// Error rendered to user
type Error struct {
	Status int
	Err    error
}

// NewError create an http error
func NewError(status int, err error) *Error {
	if err == nil {
		return nil
	}

	return &Error{
		Status: status,
		Err:    err,
	}
}

// StorageItem describe item on a storage provider
type StorageItem struct {
	Pathname string
	Name     string
	IsDir    bool
	Date     time.Time

	Info interface{}
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

// RenderItem is a storage item with an id
type RenderItem struct {
	ID string
	StorageItem
}
