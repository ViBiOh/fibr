package provider

import (
	"os"

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
