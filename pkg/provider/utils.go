package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
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
			return bytes.NewBuffer(make([]byte, 4*1024))
		},
	}

	transformerPool = sync.Pool{
		New: func() interface{} {
			return transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
		},
	}

	// SlowClient allows 5 minutes timeout
	SlowClient = request.CreateClient(5*time.Minute, request.NoRedirection)
)

// Join concatenates strings respecting terminal slashes
func Join(parts ...string) string {
	pathname := path.Join(parts...)

	for i := len(parts) - 1; i >= 0; i-- {
		if part := parts[i]; len(part) != 0 {
			if strings.HasSuffix(part, "/") && !strings.HasSuffix(pathname, "/") {
				pathname += "/"
			}
			break
		}
	}

	return pathname
}

// Dirname ensures given name is a dirname, with a trailing slash
func Dirname(name string) string {
	if !strings.HasSuffix(name, "/") {
		return name + "/"
	}
	return name
}

// GetPathname computes pathname for given params
func GetPathname(folder, name string, share Share) string {
	return Join("/", share.Path, folder, name)
}

// URL computes URL for given params
func URL(folder, name string, share Share) string {
	return Join("/", share.ID, folder, name)
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

// DoneWriter writes content to writer with error handling and done checking
func DoneWriter(isDone func() bool, w io.Writer, content string) {
	if isDone() {
		return
	}

	SafeWrite(w, content)
}

// FindPath finds index of given value into array, or -1 if not found
func FindPath(arr []string, value string) int {
	for index, item := range arr {
		if strings.HasPrefix(item, value) {
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

// LoadJSON loads JSON content
func LoadJSON(ctx context.Context, storageApp absto.Storage, filename string, content interface{}) (err error) {
	var reader io.ReadCloser
	reader, err = storageApp.ReadFrom(ctx, filename)
	if err != nil {
		return fmt.Errorf("unable to read: %w", err)
	}

	defer func() {
		err = HandleClose(reader, err)
	}()

	if err = json.NewDecoder(reader).Decode(content); err != nil {
		err = fmt.Errorf("unable to decode: %w", err)
	}

	return
}

// SaveJSON saves JSON content
func SaveJSON(ctx context.Context, storageApp absto.Storage, filename string, content interface{}) error {
	reader, writer := io.Pipe()

	done := make(chan error)
	go func() {
		defer close(done)
		var err error

		if jsonErr := json.NewEncoder(writer).Encode(content); jsonErr != nil {
			err = fmt.Errorf("unable to encode: %w", jsonErr)
		}

		if closeErr := writer.Close(); closeErr != nil {
			err = model.WrapError(err, fmt.Errorf("unable to close encoder: %w", closeErr))
		}

		done <- err
	}()

	err := storageApp.WriteTo(ctx, filename, reader)

	if jsonErr := <-done; jsonErr != nil {
		err = model.WrapError(err, jsonErr)
	}

	return err
}

// SendLargeFile in a request with buffered copy
func SendLargeFile(ctx context.Context, storageApp absto.Storage, item absto.Item, req request.Request) (*http.Response, error) {
	file, err := storageApp.ReadFrom(ctx, item.Pathname) // will be closed by `PipeWriter`
	if err != nil {
		return nil, fmt.Errorf("unable to get reader: %w", err)
	}

	reader, writer := io.Pipe()
	go func() {
		defer LogClose(file, "provider.SendLargeFile", item.Pathname)

		buffer := BufferPool.Get().(*bytes.Buffer)
		defer BufferPool.Put(buffer)

		var err error
		if _, err = io.CopyBuffer(writer, file, buffer.Bytes()); err != nil {
			err = fmt.Errorf("unable to copy: %w", err)
		}

		_ = writer.CloseWithError(err)
	}()

	r, err := req.Build(ctx, reader)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %s", err)
	}

	r.ContentLength = item.Size

	return request.DoWithClient(SlowClient, r)
}

// HandleClose closes given closer respecting given err
func HandleClose(closer io.Closer, err error) error {
	if closeErr := closer.Close(); closeErr != nil {
		return model.WrapError(err, fmt.Errorf("unable to close: %s", closeErr))
	}

	return err
}

// LogClose closes given closer and logging in case of error
func LogClose(closer io.Closer, fn, item string) {
	if err := closer.Close(); err != nil {
		logger.WithField("fn", "fn").WithField("item", item).Error("unable to close: %s", err)
	}
}

// WriteToStorage writes given content to storage
func WriteToStorage(ctx context.Context, storageApp absto.Storage, output string, reader io.Reader) error {
	directory := path.Dir(output)
	if _, err := storageApp.Info(ctx, directory); absto.IsNotExist(err) {
		if err := storageApp.CreateDir(ctx, directory); err != nil {
			return fmt.Errorf("unable to create directory: %s", err)
		}
	}

	err := storageApp.WriteTo(ctx, output, reader)
	if err != nil {
		if removeErr := storageApp.Remove(ctx, output); removeErr != nil {
			err = model.WrapError(err, fmt.Errorf("unable to remove: %s", removeErr))
		}
	}

	return err
}

// EtagMatch check that given hash match the existing etag
func EtagMatch(w http.ResponseWriter, r *http.Request, hash string) (string, bool) {
	etag := fmt.Sprintf(`W/"%s"`, hash)

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return etag, true
	}

	return etag, false
}
