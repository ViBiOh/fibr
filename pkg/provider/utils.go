package provider

import (
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	transformer  transform.Transformer
	specialChars = regexp.MustCompile(`[^a-z0-9.\-_/](?m)`)
)

func init() {
	transformer = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
}

// GetPathname computes pathname for given params
func GetPathname(folder, name string, share *Share) string {
	parts := make([]string, 0)

	if share != nil {
		parts = append(parts, share.Path)
	}

	parts = append(parts, folder)

	if len(name) > 0 {
		parts = append(parts, name)
	}

	return path.Join(parts...)
}

// GetURI computes public URI for given params
func GetURI(folder, name string, share *Share) string {
	parts := make([]string, 0)

	if share != nil {
		parts = append(parts, "/", share.ID)
	}

	parts = append(parts, folder)

	if len(name) > 0 {
		parts = append(parts, name)
	}

	return path.Join(parts...)
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

// FindIndex finds index of given value into array, or -1 if not found
func FindIndex(arr []string, value string) int {
	for index, item := range arr {
		if item == value {
			return index
		}
	}

	return -1
}

// RemoveIndex removes element at given index, if valid
func RemoveIndex(arr []string, index int) []string {
	if len(arr) == 0 || index < 0 || index >= len(arr) {
		return arr
	}

	return append(arr[:index], arr[index+1:]...)
}
