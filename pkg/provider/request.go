package provider

import (
	"fmt"
	"net/http"
	"strings"
)

var (
	// GridDisplay format
	GridDisplay = "grid"
	// ListDisplay format
	ListDisplay = "list"
	// DefaultDisplay format
	DefaultDisplay = GridDisplay

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
	Item        string
	Display     string
	Preferences Preferences
	Share       Share
	CanEdit     bool
	CanShare    bool
	CanWebhook  bool
}

// RelativeURL compute relative URL of item for that request
func (r Request) RelativeURL(item StorageItem) string {
	pathname := item.Pathname

	if !r.Share.IsZero() {
		pathname = fmt.Sprintf("/%s", strings.TrimPrefix(pathname, r.Share.Path))
	}

	if item.IsDir {
		pathname += "/"
	}

	return strings.TrimPrefix(pathname, r.Path)
}

// AbsoluteURL compute absolute URL for the given name
func (r Request) AbsoluteURL(name string) string {
	return Join("/", r.Share.ID, r.Path, name)
}

// Filepath returns the pathname of the request
func (r Request) Filepath() string {
	return r.SubPath(r.Item)
}

// SubPath returns the pathname of given name
func (r Request) SubPath(name string) string {
	return Join(r.Share.Path, r.Path, name)
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

	if len(r.Item) > 0 {
		itemPath := strings.Trim(r.Item, "/")

		if itemPath != "" {
			parts = append(parts, itemPath)
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

	if len(r.Item) > 0 {
		itemPath := strings.Trim(r.Item, "/")

		if itemPath != "" {
			parts = append(parts, itemPath)
		}
	}

	return strings.Join(parts, " - ")
}

func computeListLayoutPaths(request Request) string {
	listLayoutPaths := request.Preferences.ListLayoutPath
	path := request.Path

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
