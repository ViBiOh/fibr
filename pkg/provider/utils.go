package provider

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	transformer  transform.Transformer
	specialChars = regexp.MustCompile(`[^a-z0-9.\-_/](?m)`)
)

func init() {
	transformer = transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r)
	}), norm.NFC)
}

// SanitizeName return sanitized name (remove diacritics)
func SanitizeName(name string, removeSlash bool) (string, error) {
	withoutDiacritics, _, err := transform.String(transformer, strings.ToLower(name))
	if err != nil {
		return "", err
	}

	withoutSpaces := strings.Replace(withoutDiacritics, " ", "_", -1)
	withoutSpecials := specialChars.ReplaceAllString(withoutSpaces, "")

	sanitized := withoutSpecials
	if removeSlash {
		sanitized = strings.Replace(sanitized, "/", "_", -1)
	}

	return sanitized, nil
}

// SafeWrite writes content to writer with error handling
func SafeWrite(w io.Writer, content string) {
	if _, err := io.WriteString(w, content); err != nil {
		logger.Error("unable to write content: %s", err)
	}
}

// ErrNotExist create a NotExist error
func ErrNotExist(err error) error {
	return fmt.Errorf("path not found: %w", err)
}

// IsNotExist checks if error match a not found
func IsNotExist(err error) bool {
	if err == nil {
		return false
	}

	return strings.HasPrefix(err.Error(), "path not found")
}
