package provider

import (
	"fmt"
	"net/http"
	"path"
	"strings"
)

var (
	// NoneShare is an undefined Share
	NoneShare = Share{}

	// DefaultDisplay format
	DefaultDisplay = "grid"

	ipHeaders = []string{
		"Cf-Connecting-Ip",
		"X-Forwarded-For",
		"X-Real-Ip",
	}
)

// Preferences holds preferences of the user
type Preferences struct {
	ListLayoutPath []string
}

// Request from user
type Request struct {
	Path        string
	Display     string
	SelfURL     string
	Preferences Preferences
	Share       Share
	CanEdit     bool
	CanShare    bool
	CanWebhook  bool
}

// AbsURL compute absolute URL for the given name
func (r Request) AbsURL(name string) string {
	return path.Join(r.SelfURL, name)
}

func (r Request) relativePath(item StorageItem) string {
	if !r.Share.IsZero() {
		return fmt.Sprintf("/%s", strings.TrimPrefix(item.Pathname, r.Share.Path))
	}

	return item.Pathname
}

// RelativeURL compute relative URL of item for that request
func (r Request) RelativeURL(item StorageItem) string {
	return strings.TrimPrefix(r.relativePath(item), r.Path)
}

// RootPath compute root path of an item for that request.
// For a share, rootPath is the root of shared folder
func (r Request) RootPath(item StorageItem) string {
	return path.Dir(r.relativePath(item))
}

// GetFilepath of request
func (r Request) GetFilepath(name string) string {
	pathname := GetPathname(r.Path, name, r.Share)

	if len(name) == 0 && strings.HasSuffix(r.Path, "/") && !r.Share.File {
		return Dirname(pathname)
	}

	return pathname
}

// Layout returns layout of given name based on preferences
func (r Request) Layout(name string) string {
	return r.LayoutPath(r.AbsURL(name))
}

// LayoutPath returns layout of given path based on preferences
func (r Request) LayoutPath(path string) string {
	if FindIndex(r.Preferences.ListLayoutPath, path) != -1 {
		return "list"
	}
	return DefaultDisplay
}

// Title returns title of the page
func (r Request) Title() string {
	parts := []string{"fibr"}

	if !r.Share.IsZero() {
		parts = append(parts, r.Share.RootName)
	}

	if len(r.Path) > 0 {
		requestPath := strings.Trim(r.Path, "/")

		if requestPath != "" {
			parts = append(parts, requestPath)
		}
	}

	return strings.Join(parts, " - ")
}

// Description returns description of the page
func (r Request) Description() string {
	parts := []string{"FIle BRowser"}

	if !r.Share.IsZero() {
		parts = append(parts, r.Share.RootName)
	}

	if len(r.Path) > 0 {
		requestPath := strings.Trim(r.Path, "/")

		if requestPath != "" {
			parts = append(parts, requestPath)
		}
	}

	return strings.Join(parts, " - ")
}

func computeListLayoutPaths(request Request) string {
	listLayoutPaths := request.Preferences.ListLayoutPath
	path := request.SelfURL

	switch request.Display {
	case "list":
		if index := FindIndex(listLayoutPaths, path); index == -1 {
			listLayoutPaths = append(listLayoutPaths, path)
		}
	case DefaultDisplay:
		listLayoutPaths = RemoveIndex(listLayoutPaths, FindIndex(listLayoutPaths, path))
	}

	return strings.Join(listLayoutPaths, ",")
}

// SetPrefsCookie set preferences cookie for given request
func SetPrefsCookie(w http.ResponseWriter, request Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "list_layout_paths",
		Value:    computeListLayoutPaths(request),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Add("content-language", "en")
}

// GetIP retrieves request original IP
func GetIP(r *http.Request) string {
	for _, header := range ipHeaders {
		if ip := r.Header.Get(header); len(ip) != 0 {
			return ip
		}
	}

	return r.RemoteAddr
}
