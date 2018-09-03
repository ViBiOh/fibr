package provider

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/ViBiOh/fibr/pkg/utils"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	transformer  = getTransformer()
	specialChars = regexp.MustCompile(`[[\](){}&"'§!$*€^%+=/\\;?\x60](?m)`)
)

func getTransformer() transform.Transformer {
	// From https://blog.golang.org/normalization

	return transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r)
	}), norm.NFC)
}

// SanitizeName return sanitized name (remove diacritics)
func SanitizeName(name string) (string, error) {
	withouDiacritics, _, err := transform.String(transformer, name)
	if err != nil {
		return ``, fmt.Errorf(`Error while transforming string: %v`, err)
	}

	withoutSpecials := specialChars.ReplaceAllString(withouDiacritics, ``)
	withoutSpaces := strings.Replace(withoutSpecials, ` `, `_`, -1)
	toLower := strings.ToLower(withoutSpaces)

	return toLower, nil
}

// GetFileinfoFromRoot return file informations from given paths
func GetFileinfoFromRoot(root string, request *Request, name []byte) (string, os.FileInfo) {
	paths := []string{root}

	if request != nil {
		if request.Share != nil {
			paths = append(paths, request.Share.Path)
		}

		paths = append(paths, request.Path)
	}

	if name != nil {
		paths = append(paths, string(name))
	}

	return utils.GetPathInfo(paths...)
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
