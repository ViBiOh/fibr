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
	if err := bcrypt.CompareHashAndPassword([]byte(s.Password), []byte(password)); err != nil {
		return errors.New("invalid credentials")
	}

	return nil
}

// IsExpired check if given share is expired
func (s Share) IsExpired(now time.Time) bool {
	return s.Duration != 0 && s.Creation.Add(s.Duration).Before(now)
}
