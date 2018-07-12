package provider

import (
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/utils"
)

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
func GetPathname(request *Request, name []byte) string {
	paths := make([]string, 0)

	if request != nil {
		paths = append(paths, request.GetPath())
	}

	if name != nil {
		paths = append(paths, string(name))
	}

	return path.Join(paths...)
}

// IsNotExist checks if error match a not found
func IsNotExist(err error) bool {
	return strings.HasSuffix(err.Error(), `no such file or directory`)
}
