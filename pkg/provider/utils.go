package provider

import (
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/ViBiOh/httputils/pkg/errors"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	transformer  = getTransformer()
	specialChars = regexp.MustCompile(`[[\](){}&"'§!$*€^%+=\\;?\x60](?m)`)
)

func getTransformer() transform.Transformer {
	return transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r)
	}), norm.NFC)
}

// SanitizeName return sanitized name (remove diacritics)
func SanitizeName(name string, removeSlash bool) (string, error) {
	withoutDiacritics, _, err := transform.String(transformer, name)
	if err != nil {
		return ``, errors.WithStack(err)
	}

	withoutSpecials := specialChars.ReplaceAllString(withoutDiacritics, ``)
	withoutSpaces := strings.Replace(withoutSpecials, ` `, `_`, -1)
	toLower := strings.ToLower(withoutSpaces)

	sanitized := toLower
	if removeSlash {
		sanitized = strings.Replace(sanitized, `/`, `_`, -1)
	}

	return sanitized, nil
}

// GetPathname return file path from given paths
func GetPathname(request *Request, name string) string {
	paths := make([]string, 0)

	if request != nil {
		paths = append(paths, request.GetPath())
	}

	if name != `` {
		paths = append(paths, name)
	}

	return path.Join(paths...)
}

// IsNotExist checks if error match a not found
func IsNotExist(err error) bool {
	return strings.HasSuffix(err.Error(), `no such file or directory`)
}
