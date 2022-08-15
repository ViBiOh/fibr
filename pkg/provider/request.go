package provider

import (
	"fmt"
	"net/http"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
)

var (
	// GridDisplay format
	GridDisplay = "grid"
	// ListDisplay format
	ListDisplay = "list"
	// StoryDisplay format
	StoryDisplay = "story"

	// DefaultDisplay format
	DefaultDisplay = GridDisplay

	// LayoutPathsCookieName for saving preferences
	LayoutPathsCookieName = "layout_paths"

	preferencesPathSeparator = "|"
	ipHeaders                = []string{
		"Cf-Connecting-Ip",
		"X-Forwarded-For",
		"X-Real-Ip",
	}
)

type Preferences struct {
	LayoutPaths map[string]string
}

func ParsePreferences(value string) Preferences {
	var output Preferences

	if len(value) == 0 {
		return output
	}

	output.LayoutPaths = make(map[string]string)

	for _, part := range strings.Split(value, ",") {
		parts := strings.SplitN(part, preferencesPathSeparator, 2)
		if len(parts) == 2 {
			output.LayoutPaths[parts[0]] = parts[1]
		}
	}

	return output
}

func (p Preferences) AddLayout(path, display string) Preferences {
	if p.LayoutPaths == nil {
		p.LayoutPaths = map[string]string{
			path: display,
		}
		return p
	}

	p.LayoutPaths[path] = display
	return p
}

func (p Preferences) RemoveLayout(path string) Preferences {
	delete(p.LayoutPaths, path)

	return p
}

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

func (r Request) UpdatePreferences() Request {
	if r.Display == DefaultDisplay {
		r.Preferences = r.Preferences.RemoveLayout(r.AbsoluteURL(""))
	} else {
		r.Preferences = r.Preferences.AddLayout(r.AbsoluteURL(""), r.Display)
	}

	return r
}

func (r Request) DeletePreference(path string) Request {
	r.Preferences.RemoveLayout(path)
	return r
}

func (r Request) RelativeURL(item absto.Item) string {
	pathname := item.Pathname

	if !r.Share.IsZero() {
		pathname = fmt.Sprintf("/%s", strings.TrimPrefix(pathname, r.Share.Path))
	}

	if item.IsDir {
		pathname = Dirname(pathname)
	}

	return strings.TrimPrefix(pathname, r.Path)
}

func (r Request) IsStory() bool {
	return r.Display == StoryDisplay
}

func (r Request) AbsoluteURL(name string) string {
	return Join("/", r.Share.ID, r.Path, name)
}

func (r Request) Filepath() string {
	return r.SubPath(r.Item)
}

func (r Request) SubPath(name string) string {
	return Join(r.Share.Path, r.Path, name)
}

func (r Request) LayoutPath(path string) string {
	if layout, ok := r.Preferences.LayoutPaths[path]; ok {
		return layout
	}
	return DefaultDisplay
}

func (r Request) Title() string {
	parts := []string{"fibr"}
	parts = append(parts, r.contentParts()...)

	return strings.Join(parts, " - ")
}

func (r Request) Description() string {
	parts := []string{"FIle BRowser"}
	parts = append(parts, r.contentParts()...)

	return strings.Join(parts, " - ")
}

func (r Request) contentParts() []string {
	var parts []string

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

	return parts
}

func computeLayoutPaths(request Request) string {
	var builder strings.Builder

	for key, value := range request.Preferences.LayoutPaths {
		if builder.Len() > 0 {
			builder.WriteString(",")
		}

		builder.WriteString(key)
		builder.WriteString("|")
		builder.WriteString(value)
	}

	return builder.String()
}

func SetPrefsCookie(w http.ResponseWriter, request Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     LayoutPathsCookieName,
		Value:    computeLayoutPaths(request),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Add("content-language", "en")
}

func GetIP(r *http.Request) string {
	for _, header := range ipHeaders {
		if ip := r.Header.Get(header); len(ip) != 0 {
			return ip
		}
	}

	return r.RemoteAddr
}
