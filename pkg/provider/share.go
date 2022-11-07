package provider

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Share struct {
	Creation time.Time     `json:"creation"`
	ID       string        `json:"id"`
	Path     string        `json:"path"`
	RootName string        `json:"rootName"`
	Password string        `json:"password"`
	Duration time.Duration `json:"duration"`
	Edit     bool          `json:"edit"`
	Story    bool          `json:"story"`
	File     bool          `json:"file"`
}

func (s Share) String() string {
	var output strings.Builder

	output.WriteString(s.ID)
	output.WriteString(strconv.FormatBool(s.Edit))
	output.WriteString(s.Path)
	output.WriteString(strconv.FormatBool(s.Story))
	output.WriteString(s.RootName)
	output.WriteString(strconv.FormatBool(s.File))

	return output.String()
}

func (s Share) IsZero() bool {
	return len(s.ID) == 0
}

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

func (s Share) IsExpired(now time.Time) bool {
	return s.Duration != 0 && s.Creation.Add(s.Duration).Before(now)
}
