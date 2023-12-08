package provider

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/rs/xid"
	"github.com/zeebo/xxh3"
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
	pathEscape   = regexp.MustCompile(`\.{2,}(?m)`)

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
	withoutPathEscape := pathEscape.ReplaceAllString(withoutSpecials, "_")

	sanitized := withoutPathEscape
	if removeSlash {
		sanitized = strings.Replace(sanitized, "/", "_", -1)
	}

	return strings.Replace(sanitized, "__", "_", -1), nil
}

func SafeWrite(ctx context.Context, w io.Writer, content string) {
	if _, err := io.WriteString(w, content); err != nil {
		slog.ErrorContext(ctx, "write content", "error", err)
	}
}

func DoneWriter(ctx context.Context, isDone func() bool, w io.Writer, content string) {
	if isDone() {
		return
	}

	SafeWrite(ctx, w, content)
}

func SendLargeFile(ctx context.Context, storageService absto.Storage, item absto.Item, req request.Request) (*http.Response, error) {
	file, err := storageService.ReadFrom(ctx, item.Pathname) // will be closed by `PipeWriter`
	if err != nil {
		return nil, fmt.Errorf("get reader: %w", err)
	}

	reader, writer := io.Pipe()
	go func() {
		defer LogClose(ctx, file, "provider.SendLargeFile", item.Pathname)

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

	r.ContentLength = item.Size()

	return request.DoWithClient(SlowClient, r)
}

func LogClose(ctx context.Context, closer io.Closer, fn, item string) {
	if err := closer.Close(); err != nil {
		slog.ErrorContext(ctx, "close", "error", err, "fn", fn, "item", item)
	}
}

func WriteToStorage(ctx context.Context, storageService absto.Storage, output string, size int64, reader io.Reader) error {
	var err error
	directory := path.Dir(output)

	if err = storageService.Mkdir(ctx, directory, absto.DirectoryPerm); err != nil {
		return fmt.Errorf("create directory `%s`: %w", directory, err)
	}

	err = storageService.WriteTo(ctx, output, reader, absto.WriteOpts{Size: size})
	if err != nil {
		if removeErr := storageService.RemoveAll(ctx, output); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("remove: %w", removeErr))
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

func Identifier() string {
	return xid.New().String()
}

func Hash(value string) string {
	return strconv.FormatUint(xxh3.HashString(value), 16)
}

func RawHash(content any) string {
	hasher := xxh3.New()

	fmt.Fprintf(hasher, "%v", content)

	return hex.EncodeToString(hasher.Sum(nil))
}
