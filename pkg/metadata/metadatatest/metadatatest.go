package metadatatest

import (
	"time"

	"github.com/ViBiOh/fibr/pkg/metadata"
	"github.com/ViBiOh/fibr/pkg/provider"
	"golang.org/x/crypto/bcrypt"
)

var _ metadata.App = &App{}

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

// App mocks implementation
type App struct {
	enabled bool

	getShare provider.Share

	createShareID    string
	createShareError error

	renameSharePath error

	deleteShare error

	deleteSharePath error

	dump map[string]provider.Share
}

// New creates new mocked instance
func New() *App {
	return &App{}
}

// SetEnabled mocks implementation
func (a *App) SetEnabled(value bool) *App {
	a.enabled = value

	return a
}

// SetGetShare mocks implementation
func (a *App) SetGetShare(share provider.Share) *App {
	a.getShare = share

	return a
}

// SetCreateShare mocks implementation
func (a *App) SetCreateShare(id string, err error) *App {
	a.createShareID = id
	a.createShareError = err

	return a
}

// SetRenameSharePath mocks implementation
func (a *App) SetRenameSharePath(err error) *App {
	a.renameSharePath = err

	return a
}

// SetDeleteShare mocks implementation
func (a *App) SetDeleteShare(err error) *App {
	a.deleteShare = err

	return a
}

// SetDeleteSharePath mocks implementation
func (a *App) SetDeleteSharePath(err error) *App {
	a.deleteSharePath = err

	return a
}

// SetDump mocks implementation
func (a *App) SetDump(shares map[string]provider.Share) *App {
	a.dump = shares

	return a
}

// Enabled mocks implementation
func (a *App) Enabled() bool {
	return a.enabled
}

// GetShare mocks implementation
func (a *App) GetShare(string) provider.Share {
	return a.getShare
}

// CreateShare mocks implementation
func (a *App) CreateShare(string, bool, string, bool, time.Duration) (string, error) {
	return a.createShareID, a.createShareError
}

// RenameSharePath mocks implementation
func (a *App) RenameSharePath(string, string) error {
	return a.renameSharePath
}

// DeleteShare mocks implementation
func (a *App) DeleteShare(string) error {
	return a.deleteShare
}

// DeleteSharePath mocks implementation
func (a *App) DeleteSharePath(string) error {
	return a.deleteSharePath
}

// Dump mocks implementation
func (a *App) Dump() map[string]provider.Share {
	return a.dump
}

// Start mocks implementation
func (a *App) Start(<-chan struct{}) {
	return
}
