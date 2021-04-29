package crudtest

import (
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"golang.org/x/crypto/bcrypt"
)

var (
	// PasswordLessShare instance
	PasswordLessShare = provider.Share{
		ID:       "a1b2c3d4f5",
		Edit:     false,
		RootName: "public",
		File:     false,
		Path:     "/public",
	}

	passwordHash, _ = bcrypt.GenerateFromPassword([]byte("password"), 12)

	// PasswordShare instance
	PasswordShare = provider.Share{
		ID:       "f5d4c3b2a1",
		Edit:     true,
		RootName: "private",
		File:     false,
		Path:     "/private",
		Password: string(passwordHash),
	}
)

// App for mocked calls
type App struct{}

// New creates new mocked instance
func New() App {
	return App{}
}

// Start mocked implementation
func (a App) Start(<-chan struct{}) {
	// mock implementation
}

// Browser mocked implementation
func (a App) Browser(http.ResponseWriter, provider.Request, provider.StorageItem, renderer.Message) {
	// mock implementation
}

// ServeStatic mocked implementation
func (a App) ServeStatic(http.ResponseWriter, *http.Request) bool {
	return false
}

// List mocked implementation
func (a App) List(http.ResponseWriter, provider.Request, renderer.Message) {
	// mock implementation
}

// Get mocked implementation
func (a App) Get(http.ResponseWriter, *http.Request, provider.Request) {
	// mock implementation
}

// Post mocked implementation
func (a App) Post(http.ResponseWriter, *http.Request, provider.Request) {
	// mock implementation
}

// Create mocked implementation
func (a App) Create(http.ResponseWriter, *http.Request, provider.Request) {
	// mock implementation
}

// Upload mocked implementation
func (a App) Upload(http.ResponseWriter, *http.Request, provider.Request, map[string]string, *multipart.Part) {
	// mock implementation
}

// Rename mocked implementation
func (a App) Rename(http.ResponseWriter, *http.Request, provider.Request) {
	// mock implementation
}

// Delete mocked implementation
func (a App) Delete(http.ResponseWriter, *http.Request, provider.Request) {
	// mock implementation
}

// GetShare mocked implementation
func (a App) GetShare(path string) provider.Share {
	if strings.HasPrefix(path, "/a1b2c3d4f5") {
		return PasswordLessShare
	}

	if strings.HasPrefix(path, "/f5d4c3b2a1") {
		return PasswordShare
	}

	return provider.Share{}
}

// CreateShare mocked implementation
func (a App) CreateShare(http.ResponseWriter, *http.Request, provider.Request) {
	// mock implementation
}

// DeleteShare mocked implementation
func (a App) DeleteShare(http.ResponseWriter, *http.Request, provider.Request) {
	// mock implementation
}
