package provider

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Share stores informations about shared paths
type Share struct {
	Creation time.Time     `json:"creation"`
	ID       string        `json:"id"`
	Path     string        `json:"path"`
	RootName string        `json:"rootName"`
	Password string        `json:"password"`
	Duration time.Duration `json:"duration"`
	Edit     bool          `json:"edit"`
	File     bool          `json:"file"`
}

// IsZero verifies that instance is hydrated
func (s Share) IsZero() bool {
	return len(s.ID) == 0
}

// CheckPassword verifies that request has correct password for share
func (s Share) CheckPassword(authorizationHeader string) error {
	if s.Password == "" {
		return nil
	}

	if authorizationHeader == "" {
		return errors.New("empty authorization header")
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authorizationHeader, "Basic "))
	if err != nil {
		return err
	}

	dataStr := string(data)

	sepIndex := strings.Index(dataStr, ":")
	if sepIndex < 0 {
		return errors.New("invalid format for basic auth")
	}

	password := dataStr[sepIndex+1:]
	if err = bcrypt.CompareHashAndPassword([]byte(s.Password), []byte(password)); err != nil {
		return errors.New("invalid credentials")
	}

	return nil
}

// IsExpired check if given share is expired
func (s Share) IsExpired(now time.Time) bool {
	return s.Duration != 0 && s.Creation.Add(s.Duration).Before(now)
}

// ShareByID sort Share by ID
type ShareByID []Share

func (a ShareByID) Len() int      { return len(a) }
func (a ShareByID) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ShareByID) Less(i, j int) bool {
	return a[i].ID < a[j].ID
}
