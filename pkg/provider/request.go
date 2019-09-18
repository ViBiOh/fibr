package provider

import (
	"encoding/base64"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/ViBiOh/httputils/v2/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// Request from user
type Request struct {
	Path     string
	CanEdit  bool
	CanShare bool
	Display  string
	Order    string
	Share    *Share
}

// GetFilepath of request
func (r Request) GetFilepath(name string) string {
	parts := make([]string, 0)

	if r.Share != nil {
		parts = append(parts, r.Share.Path)
	}

	parts = append(parts, r.Path, name)

	return path.Join(parts...)
}

// GetURI of request
func (r Request) GetURI(name string) string {
	parts := make([]string, 0)

	if r.Share != nil {
		parts = append(parts, r.Share.ID)
	}

	parts = append(parts, r.Path, name)

	return path.Join(parts...)
}

// Share stores informations about shared paths
type Share struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	RootName string `json:"rootName"`
	Edit     bool   `json:"edit"`
	Password string `json:"password"`
}

// CheckPassword verifies that request has correct password for share
func (s Share) CheckPassword(r *http.Request) error {
	if s.Password == "" {
		return nil
	}

	header := r.Header.Get("Authorization")
	if header == "" {
		return errors.New("empty authorization header")
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		return errors.WithStack(err)
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
	if mime := mime.TypeByExtension(extension); mime != "" {
		return mime
	}

	if CodeExtensions[extension] {
		return "text/plain; charset=utf-8"
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
