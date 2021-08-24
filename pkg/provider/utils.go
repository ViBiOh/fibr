package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
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

	transformerPool = sync.Pool{
		New: func() interface{} {
			return transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
		},
	}
)

// GetPathname computes pathname for given params
func GetPathname(folder, name string, share Share) string {
	return filepath.Join(buildPath(folder, name, share.Path)...)
}

// URL computes public URI for given params
func URL(folder, name string, share Share) string {
	return path.Join(buildPath(folder, name, share.ID)...)
}

func buildPath(folder, name, share string) []string {
	parts := []string{"/"}

	if len(share) > 0 {
		parts = append(parts, share)
	}

	parts = append(parts, folder)

	if len(name) > 0 {
		parts = append(parts, name)
	}

	return parts
}

// SanitizeName return sanitized name (remove diacritics)
func SanitizeName(name string, removeSlash bool) (string, error) {
	withoutLigatures := strings.ToLower(name)
	for key, value := range transliterations {
		if strings.Contains(withoutLigatures, key) {
			withoutLigatures = strings.ReplaceAll(withoutLigatures, key, value)
		}
	}

	transformer := transformerPool.Get().(transform.Transformer)
	defer transformerPool.Put(transformer)

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

// SendLargeFile in a request with buffered copy
func SendLargeFile(ctx context.Context, storageApp Storage, item StorageItem, client *http.Client, url string) (*http.Response, error) {
	file, err := storageApp.ReaderFrom(item.Pathname) // will be closed by `.PipedWriter`
	if err != nil {
		return nil, fmt.Errorf("unable to get reader: %s", err)
	}

	reader, writer := io.Pipe()
	go func() {
		buffer := BufferPool.Get().(*bytes.Buffer)
		defer BufferPool.Put(buffer)

		if _, err := io.CopyBuffer(writer, file, buffer.Bytes()); err != nil {
			logger.Error("unable to copy file: %s", err)
		}

		_ = writer.CloseWithError(file.Close())
	}()

	r, err := request.New().Post(url).Build(ctx, reader)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %s", err)
	}

	r.ContentLength = item.Size

	return request.DoWithClient(client, r)
}
