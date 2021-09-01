package crud

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

type entry struct {
	Key   string
	Value interface{}
}

// Stats render stats of the current
func (a App) Stats(w http.ResponseWriter, request provider.Request, message renderer.Message) (string, int, map[string]interface{}, error) {
	stats, err := a.computeStats()
	if err != nil {
		return "", http.StatusInternalServerError, nil, err
	}

	return "stats", http.StatusOK, map[string]interface{}{
		"Request": request,
		"Message": message,
		"Stats": []entry{
			{Key: "Directories", Value: stats["Directories"]},
			{Key: "Files", Value: stats["Files"]},
			{Key: "Size", Value: fmt.Sprintf("%.2f MB", float64(stats["Size"])/1024/1024)},
			{Key: "Metadatas", Value: fmt.Sprintf("%.2f MB", float64(stats["Metadatas"])/1024/1024)},
		},
	}, nil
}

func (a App) computeStats() (map[string]uint64, error) {
	var filesCount, directoriesCount, filesSize, metadataSize uint64

	err := a.storageApp.Walk("", func(item provider.StorageItem, err error) error {
		if err != nil {
			return err
		}

		if item.IsDir {
			directoriesCount++
		} else {
			filesCount++
			filesSize += uint64(item.Size)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("unable to browse files: %s", err)
	}

	err = a.rawStorageApp.Walk(provider.MetadataDirectoryName, func(item provider.StorageItem, err error) error {
		if err != nil {
			return err
		}

		if !item.IsDir {
			metadataSize += uint64(item.Size)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("unable to browse metadatas: %s", err)
	}

	return map[string]uint64{
		"Files":       filesCount,
		"Directories": directoriesCount,
		"Size":        filesSize,
		"Metadatas":   metadataSize,
	}, nil
}
