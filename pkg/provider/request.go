package provider

import (
	"fmt"
	"net/http"
	"strings"
)

var (
	// NoneShare is an undefined Share
	NoneShare = Share{}
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

// GetURI of request
func (r Request) GetURI(name string) string {
	return GetURI(r.Path, name, r.Share)
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

func computeListLayoutPaths(request Request) string {
	listLayoutPaths := request.Preferences.ListLayoutPath
	path := strings.Trim(request.GetURI(""), "/")

	switch request.Display {
	case "list":
		if index := FindIndex(listLayoutPaths, path); index == -1 {
			listLayoutPaths = append(listLayoutPaths, path)
		}
	case "grid":
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
		SameSite: http.SameSiteStrictMode,
	})
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

// Error rendered to user
type Error struct {
	Err    error
	Status int
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

// Error convert error to string
func (e Error) Error() string {
	return fmt.Sprintf("HTTP/%d: %s", e.Status, e.Err)
}
