package provider

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/ViBiOh/httputils/v2/pkg/errors"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	transformer  transform.Transformer
	specialChars = regexp.MustCompile(`[[\](){}&"'§!$*€^%+=\\;?\x60](?m)`)
)

func init() {
	transformer = transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r)
	}), norm.NFC)
}

// SanitizeName return sanitized name (remove diacritics)
func SanitizeName(name string, removeSlash bool) (string, error) {
	withoutDiacritics, _, err := transform.String(transformer, name)
	if err != nil {
		return "", errors.WithStack(err)
	}

	withoutSpecials := specialChars.ReplaceAllString(withoutDiacritics, "")
	withoutSpaces := strings.Replace(withoutSpecials, " ", "_", -1)
	toLower := strings.ToLower(withoutSpaces)

	sanitized := toLower
	if removeSlash {
		sanitized = strings.Replace(sanitized, "/", "_", -1)
	}

	return sanitized, nil
}

// ErrNotExist create a NotExist error
func ErrNotExist(err error) error {
	return errors.Wrap(err, "path not found")
}

// IsNotExist checks if error match a not found
func IsNotExist(err error) bool {
	if err == nil {
		return false
	}

	return err.Error() == "path not found"
}
