package utils

import (
	"os"
	"path"
)

// GetPathInfo retrieve filesystem informations for given paths
func GetPathInfo(parts ...string) (string, os.FileInfo) {
	fullPath := path.Join(parts...)
	info, err := os.Stat(fullPath)

	if err != nil {
		return fullPath, nil
	}
	return fullPath, info
}
