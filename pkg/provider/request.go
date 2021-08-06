package provider

import (
	"net/http"
	"strings"
)

var (
	// NoneShare is an undefined Share
	NoneShare = Share{}

	// DefaultDisplay format
	DefaultDisplay = "grid"
)

// Preferences holds preferences of the user
type Preferences struct {
	ListLayoutPath []string
}

// Request from user
type Request struct {
	Path        string
	Display     string
	Preferences Preferences
	Share       Share
	CanEdit     bool
	CanShare    bool
}

// GetFilepath of request
func (r Request) GetFilepath(name string) string {
	return GetPathname(r.Path, name, r.Share)
}

// URL of request
func (r Request) URL(name string) string {
	return URL(r.Path, name, r.Share)
}

// Layout returns layout of given name based on preferences
func (r Request) Layout(name string) string {
	return r.LayoutPath(r.URL(name))
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

	if len(r.Share.ID) != 0 {
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

	if len(r.Share.ID) != 0 {
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
	path := request.URL("")

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
