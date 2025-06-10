package push

import (
	"bytes"
	"encoding/base64"
	"strings"
)

type Subscription struct {
	Endpoint  string `json:"endpoint"`
	PublicKey string `json:"publicKey"`
	Auth      string `json:"auth"`
}

func (s Subscription) decodedPublicKey() ([]byte, error) {
	return decodeKey(s.PublicKey)
}

func (s Subscription) decodedAuth() ([]byte, error) {
	return decodeKey(s.Auth)
}

func decodeKey(key string) ([]byte, error) {
	buffer := bytes.NewBufferString(key)
	if rem := len(key) % 4; rem != 0 {
		buffer.WriteString(strings.Repeat("=", 4-rem))
	}

	if bytes, err := base64.StdEncoding.DecodeString(buffer.String()); err == nil {
		return bytes, nil
	}

	return base64.URLEncoding.DecodeString(buffer.String())
}

type Notification struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Image       string `json:"image"`
}
