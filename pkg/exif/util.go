package exif

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func getExifPath(item provider.StorageItem, suffix string) string {
	fullPath := path.Join(provider.MetadataDirectoryName, item.Pathname)
	if item.IsDir {
		fullPath += "/"
	}

	name := strings.TrimSuffix(fullPath, path.Ext(fullPath))

	if len(suffix) == 0 {
		return fmt.Sprintf("%s.json", name)
	}

	return fmt.Sprintf("%s_%s.json", name, suffix)
}

func (a app) saveMetadata(item provider.StorageItem, suffix string, data interface{}) error {
	writer, err := a.storageApp.WriterTo(getExifPath(item, suffix))
	if err != nil {
		return fmt.Errorf("unable to get writer: %s", err)
	}

	defer func() {
		if err := writer.Close(); err != nil {
			logger.Error("unable to close file: %s", err)
		}
	}()

	if err := json.NewEncoder(writer).Encode(data); err != nil {
		return fmt.Errorf("unable to encode: %s", err)
	}

	return nil
}
