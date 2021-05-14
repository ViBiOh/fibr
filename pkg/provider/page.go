package provider

import (
	"path"
	"regexp"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

var (
	protocolRegex = regexp.MustCompile("^(https?):/")
)

// Page renderer to user
type Page struct {
	Content     map[string]interface{}
	Error       *Error
	Message     renderer.Message
	PublicURL   string
	Title       string
	Description string
	Layout      string
	Config      Config
	Request     Request
}

// PageBuilder for interactively create page
type PageBuilder struct {
	content map[string]interface{}
	error   *Error
	message renderer.Message
	layout  string
	config  Config
	request Request
}

// Config set Config for page
func (p *PageBuilder) Config(config Config) *PageBuilder {
	p.config = config

	return p
}

// Request set Request for page
func (p *PageBuilder) Request(request Request) *PageBuilder {
	p.request = request

	return p
}

// Message set Message for page
func (p *PageBuilder) Message(message renderer.Message) *PageBuilder {
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
	publicURL := computePublicURL(p.config, p.request)
	title := computeTitle(p.config, p.request)
	description := computeDescription(p.config, p.request)

	layout := p.layout
	if layout == "" {
		layout = p.request.Layout("")
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

func computePublicURL(config Config, request Request) string {
	parts := []string{config.PublicURL}

	if len(request.Path) > 0 {
		if len(request.Share.ID) != 0 {
			parts = append(parts, request.Share.ID)
		}

		parts = append(parts, request.Path)
	}

	return protocolRegex.ReplaceAllString(path.Join(parts...), "$1://")
}

func computeTitle(config Config, request Request) string {
	parts := make([]string, 0)

	if len(config.Seo.Title) > 0 {
		parts = append(parts, config.Seo.Title)
	}

	if len(request.Share.ID) != 0 {
		parts = append(parts, request.Share.RootName)
	}

	if len(request.Path) > 0 {
		requestPath := strings.Trim(request.Path, "/")

		if requestPath != "" {
			parts = append(parts, requestPath)
		}
	}

	return strings.Join(parts, " - ")
}

func computeDescription(config Config, request Request) string {
	parts := make([]string, 0)

	if len(config.Seo.Description) > 0 {
		parts = append(parts, config.Seo.Description)
	}

	if len(request.Share.ID) != 0 {
		parts = append(parts, request.Share.RootName)
	}

	if len(request.Path) > 0 {
		requestPath := strings.Trim(request.Path, "/")

		if requestPath != "" {
			parts = append(parts, requestPath)
		}
	}

	return strings.Join(parts, " - ")
}
