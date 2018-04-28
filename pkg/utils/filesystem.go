package utils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
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

	if err := filepath.Walk(dir, func(walkedPath string, info os.FileInfo, _ error) error {
		if path.Ext(info.Name()) == ext {
			output = append(output, walkedPath)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf(`Error while listing files: %v`, err)
	}

	return output, nil
}
