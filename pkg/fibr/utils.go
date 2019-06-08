package fibr

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

func checkSharePassword(r *http.Request, share *provider.Share) error {
	header := r.Header.Get("Authorization")
	if header == "" {
		return errEmptyAuthorizationHeader
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		return errors.WithStack(err)
	}

	dataStr := string(data)

	sepIndex := strings.Index(dataStr, ":")
	if sepIndex < 0 {
		return errors.New("invalid format for basic auth")
	}

	password := dataStr[sepIndex+1:]
	if err := bcrypt.CompareHashAndPassword([]byte(share.Password), []byte(password)); err != nil {
		return errors.New("invalid credentials")
	}

	return nil
}
