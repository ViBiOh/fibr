package utils

import (
	"os"
	"path"
	"path/filepath"

	"github.com/ViBiOh/httputils/pkg/errors"
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

// ListFilesByExt list files by extension
func ListFilesByExt(dir, ext string) ([]string, error) {
	output := make([]string, 0)

	err := filepath.Walk(dir, func(walkedPath string, info os.FileInfo, _ error) error {
		if path.Ext(info.Name()) == ext {
			output = append(output, walkedPath)
		}

		return nil
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return output, nil
}
