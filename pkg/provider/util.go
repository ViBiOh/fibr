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

	BufferPool = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 4*1024))
		},
	}

	transformerPool = sync.Pool{
		New: func() any {
			return transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
		},
	}

	SlowClient = request.CreateClient(5*time.Minute, request.NoRedirection)
)

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

func Dirname(name string) string {
	if !strings.HasSuffix(name, "/") {
		return name + "/"
	}
	return name
}

func GetPathname(folder, name string, share Share) string {
	return Join("/", share.Path, folder, name)
}

func URL(folder, name string, share Share) string {
	return Join("/", share.ID, folder, name)
}

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

func SafeWrite(w io.Writer, content string) {
	if _, err := io.WriteString(w, content); err != nil {
		logger.Error("write content: %s", err)
	}
}

func DoneWriter(isDone func() bool, w io.Writer, content string) {
	if isDone() {
		return
	}

	SafeWrite(w, content)
}

func FindPath(arr []string, value string) int {
	for index, item := range arr {
		if strings.HasPrefix(item, value) {
			return index
		}
	}

	return -1
}

func RemoveIndex(arr []string, index int) []string {
	if len(arr) == 0 || index < 0 || index >= len(arr) {
		return arr
	}

	return append(arr[:index], arr[index+1:]...)
}

func LoadJSON[T any](ctx context.Context, storageApp absto.Storage, filename string) (output T, err error) {
	var reader io.ReadCloser
	reader, err = storageApp.ReadFrom(ctx, filename)
	if err != nil {
		err = fmt.Errorf("read: %w", err)
		return
	}

	defer func() {
		err = HandleClose(reader, err)
	}()

	if err = json.NewDecoder(reader).Decode(&output); err != nil {
		err = fmt.Errorf("decode: %w", storageApp.ConvertError(err))
	}

	return
}

func SaveJSON(ctx context.Context, storageApp absto.Storage, filename string, content any) error {
	reader, writer := io.Pipe()

	done := make(chan error)
	go func() {
		defer close(done)
		var err error

		if jsonErr := json.NewEncoder(writer).Encode(content); jsonErr != nil {
			err = fmt.Errorf("encode: %w", jsonErr)
		}

		if closeErr := writer.Close(); closeErr != nil {
			err = model.WrapError(err, fmt.Errorf("close encoder: %w", closeErr))
		}

		done <- err
	}()

	err := storageApp.WriteTo(ctx, filename, reader, absto.WriteOpts{})

	if jsonErr := <-done; jsonErr != nil {
		err = model.WrapError(err, jsonErr)
	}

	return err
}

func SendLargeFile(ctx context.Context, storageApp absto.Storage, item absto.Item, req request.Request) (*http.Response, error) {
	file, err := storageApp.ReadFrom(ctx, item.Pathname) // will be closed by `PipeWriter`
	if err != nil {
		return nil, fmt.Errorf("get reader: %w", err)
	}

	reader, writer := io.Pipe()
	go func() {
		defer LogClose(file, "provider.SendLargeFile", item.Pathname)

		buffer := BufferPool.Get().(*bytes.Buffer)
		defer BufferPool.Put(buffer)

		var err error
		if _, err = io.CopyBuffer(writer, file, buffer.Bytes()); err != nil {
			err = fmt.Errorf("copy: %w", err)
		}

		_ = writer.CloseWithError(err)
	}()

	r, err := req.Build(ctx, reader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	r.ContentLength = item.Size

	return request.DoWithClient(SlowClient, r)
}

func HandleClose(closer io.Closer, err error) error {
	if closeErr := closer.Close(); closeErr != nil {
		return model.WrapError(err, fmt.Errorf("close: %w", closeErr))
	}

	return err
}

func LogClose(closer io.Closer, fn, item string) {
	if err := closer.Close(); err != nil {
		logger.WithField("fn", fn).WithField("item", item).Error("close: %s", err)
	}
}

func WriteToStorage(ctx context.Context, storageApp absto.Storage, output string, size int64, reader io.Reader) error {
	var err error
	directory := path.Dir(output)

	if err = storageApp.CreateDir(ctx, directory); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	err = storageApp.WriteTo(ctx, output, reader, absto.WriteOpts{Size: size})
	if err != nil {
		if removeErr := storageApp.Remove(ctx, output); removeErr != nil {
			err = model.WrapError(err, fmt.Errorf("remove: %w", removeErr))
		}
	}

	return err
}

func EtagMatch(w http.ResponseWriter, r *http.Request, hash string) (etag string, match bool) {
	etag = fmt.Sprintf(`W/"%s"`, hash)

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		match = true
	}

	return
}

func findIndex(arr []string, value string) int {
	for index, item := range arr {
		if item == value {
			return index
		}
	}

	return -1
}
