package provider

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
)

type Display string

var (
	GridDisplay  Display = "grid"
	ListDisplay  Display = "list"
	StoryDisplay Display = "story"

	DefaultDisplay = GridDisplay

	LayoutPathsCookieName = "layout_paths"

	preferencesPathSeparator = "|"
	ipHeaders                = []string{
		"Cf-Connecting-Ip",
		"X-Forwarded-For",
		"X-Real-Ip",
	}
)

func ParseDisplay(input string) Display {
	switch input {
	case string(GridDisplay):
		return GridDisplay

	case string(ListDisplay):
		return ListDisplay

	case string(StoryDisplay):
		return StoryDisplay

	default:
		return DefaultDisplay
	}
}

type DisplayPreferences map[string]Display

func ParseDisplayPreferences(value string) map[string]Display {
	output := make(DisplayPreferences)

	for _, part := range strings.Split(value, ",") {
		parts := strings.SplitN(part, preferencesPathSeparator, 2)
		if len(parts) == 2 {
			output[parts[0]] = ParseDisplay(parts[1])
		}
	}

	return output
}

func (dp DisplayPreferences) String() string {
	var builder strings.Builder

	for key, value := range dp {
		if builder.Len() > 0 {
			builder.WriteString(",")
		}

		builder.WriteString(key)
		builder.WriteString("|")
		builder.WriteString(string(value))
	}

	return builder.String()
}

type Preferences struct {
	LayoutPaths DisplayPreferences
}

func ParsePreferences(value string) Preferences {
	var output Preferences

	if len(value) == 0 {
		return output
	}

	output.LayoutPaths = ParseDisplayPreferences(value)

	return output
}

func (p Preferences) SetCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     LayoutPathsCookieName,
		Value:    p.LayoutPaths.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func (p Preferences) GetLayout(path string) Display {
	if layout, ok := p.LayoutPaths[path]; ok {
		return layout
	}

	return DefaultDisplay
}

func (p Preferences) AddLayout(path string, display Display) Preferences {
	if p.LayoutPaths == nil {
		p.LayoutPaths = DisplayPreferences{
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
	Display     Display
	Preferences Preferences
	Share       Share
	CanEdit     bool
	CanShare    bool
	CanWebhook  bool
}

func (r Request) String() string {
	var output strings.Builder

	output.WriteString(r.Path)
	output.WriteString(strconv.FormatBool(r.CanEdit))
	output.WriteString(r.Item)
	output.WriteString(strconv.FormatBool(r.CanShare))
	output.WriteString(string(r.Display))
	output.WriteString(strconv.FormatBool(r.CanWebhook))
	output.WriteString(r.Share.String())

	return output.String()
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

func (r Request) LayoutPath(path string) Display {
	return r.Preferences.GetLayout(path)
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

func SetPrefsCookie(w http.ResponseWriter, request Request) {
	request.Preferences.SetCookie(w)

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
