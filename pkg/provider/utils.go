package provider

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	transformer      = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	transliterations = map[string]string{
		"Ð": "D",
		"Ł": "l",
		"Ø": "oe",
		"Þ": "Th",
		"ß": "ss",
		"æ": "ae",
		"ð": "d",
		"ł": "l",
		"ø": "oe",
		"þ": "th",
		"œ": "oe",
	}
	quotesChar   = regexp.MustCompile(`["'` + "`" + `](?m)`)
	specialChars = regexp.MustCompile(`[^a-z0-9.\-_/](?m)`)

	// BufferPool for io.CopyBuffer
	BufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 32*1024))
		},
	}
)

// GetPathname computes pathname for given params
func GetPathname(folder, name string, share Share) string {
	parts := make([]string, 0)

	if len(share.ID) != 0 {
		parts = append(parts, share.Path)
	}

	parts = append(parts, folder)

	if len(name) > 0 {
		parts = append(parts, name)
	}

	return path.Join(parts...)
}

// URL computes public URI for given params
func URL(folder, name string, share Share) string {
	parts := []string{"/"}

	if len(share.ID) != 0 {
		parts = append(parts, share.ID)
	}

	parts = append(parts, folder)

	if len(name) > 0 {
		parts = append(parts, name)
	}

	return path.Join(parts...)
}

// SanitizeName return sanitized name (remove diacritics)
func SanitizeName(name string, removeSlash bool) (string, error) {
	withoutLigatures := strings.ToLower(name)
	for key, value := range transliterations {
		if strings.Contains(withoutLigatures, key) {
			withoutLigatures = strings.ReplaceAll(withoutLigatures, key, value)
		}
	}

	withoutDiacritics, _, err := transform.String(transformer, withoutLigatures)
	if err != nil {
		return "", err
	}

	withoutSpaces := strings.Replace(withoutDiacritics, " ", "_", -1)
	withoutQuotes := quotesChar.ReplaceAllString(withoutSpaces, "_")
	withoutSpecials := specialChars.ReplaceAllString(withoutQuotes, "")

	sanitized := withoutSpecials
	if removeSlash {
		sanitized = strings.Replace(sanitized, "/", "_", -1)
	}

	return strings.Replace(sanitized, "__", "_", -1), nil
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
