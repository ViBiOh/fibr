package provider

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

var (
	protocolRegex = regexp.MustCompile("^(https?):/")
)

// Page renderer to user
type Page struct {
	Config  *Config
	Request *Request
	Message *Message
	Error   *Error
	Layout  string
	Content map[string]interface{}

	PublicURL   string
	Title       string
	Description string
}

// PageBuilder for interactively create page
type PageBuilder struct {
	config  *Config
	request *Request
	message *Message
	error   *Error
	layout  string
	content map[string]interface{}
}

// Config set Config for page
func (p *PageBuilder) Config(config *Config) *PageBuilder {
	p.config = config

	return p
}

// Request set Request for page
func (p *PageBuilder) Request(request *Request) *PageBuilder {
	p.request = request

	return p
}

// Message set Message for page
func (p *PageBuilder) Message(message *Message) *PageBuilder {
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
	layout := p.layout
	var publicURL, title, description string

	if p.config != nil && p.request != nil {
		publicURL = computePublicURL(p.config, p.request)
		title = computeTitle(p.config, p.request)
		description = computeDescription(p.config, p.request)
	}

	if p.layout == "" {
		layout = "grid"
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

func computePublicURL(config *Config, request *Request) string {
	parts := []string{config.PublicURL}

	if request != nil {
		if request.Share != nil {
			parts = append(parts, request.Share.ID)
		}

		parts = append(parts, request.Path)
	}

	return protocolRegex.ReplaceAllString(path.Join(parts...), "$1://")
}

func computeTitle(config *Config, request *Request) string {
	title := config.Seo.Title

	if request != nil {
		if request.Share != nil {
			title = fmt.Sprintf("%s - %s", title, request.Share.RootName)
		} else {
			title = fmt.Sprintf("%s - %s", title, config.RootName)
		}

		path := strings.Trim(request.Path, "/")
		if path != "" {
			title = fmt.Sprintf("%s - %s", title, path)
		}
	}

	return title
}

func computeDescription(config *Config, request *Request) string {
	description := config.Seo.Description

	if request != nil {
		if request.Share != nil {
			description = fmt.Sprintf("%s - %s", description, request.Share.RootName)
		} else {
			description = fmt.Sprintf("%s - %s", description, config.RootName)
		}

		if request.Path != "" {
			description = fmt.Sprintf("%s/%s", description, strings.Trim(request.Path, "/"))
		}
	}

	return description
}
