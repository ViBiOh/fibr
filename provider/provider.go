package provider

import (
	"net/http"
	"os"
)

// RequestConfig stores informations
type RequestConfig struct {
	URL        string
	Root       string
	PathPrefix string
	Path       string
	CanEdit    bool
}

// Message rendered to user
type Message struct {
	Level   string
	Content string
}

// Renderer interface for return rich content to user
type Renderer interface {
	Error(http.ResponseWriter, int, error)
	Login(http.ResponseWriter, *Message)
	Sitemap(http.ResponseWriter)
	Directory(http.ResponseWriter, *RequestConfig, []os.FileInfo, *Message)
}
