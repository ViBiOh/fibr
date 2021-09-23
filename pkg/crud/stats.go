package crud

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

type entry struct {
	Key   string
	Value interface{}
}

// Stats render stats of the current
func (a App) Stats(w http.ResponseWriter, request provider.Request, message renderer.Message) (string, int, map[string]interface{}, error) {
	pathname := request.GetFilepath("")

	stats, err := a.computeStats(pathname)
	if err != nil {
		return "", http.StatusInternalServerError, nil, err
	}

	return "stats", http.StatusOK, map[string]interface{}{
		"Request": request,
		"Message": message,
		"Stats": []entry{
			{Key: "Current path", Value: pathname},
			{Key: "Directories", Value: stats["Directories"]},
			{Key: "Files", Value: stats["Files"]},
			{Key: "Size", Value: bytesHuman(stats["Size"])},
			{Key: "Metadatas", Value: fmt.Sprintf("%s (%.1f%% of Size)", bytesHuman(stats["Metadatas"]), float64(stats["Metadatas"])/float64(stats["Size"]))},
		},
	}, nil
}

func (a App) computeStats(pathname string) (map[string]uint64, error) {
	var filesCount, directoriesCount, filesSize, metadataSize uint64

	err := a.storageApp.Walk(pathname, func(item provider.StorageItem, err error) error {
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

	metadataPath := filepath.Join(provider.MetadataDirectoryName, pathname)
	if _, err := a.rawStorageApp.Info(metadataPath); err == nil {
		err = a.rawStorageApp.Walk(metadataPath, func(item provider.StorageItem, err error) error {
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
	}

	return map[string]uint64{
		"Files":       filesCount,
		"Directories": directoriesCount,
		"Size":        filesSize,
		"Metadatas":   metadataSize,
	}, nil
}

var (
	bytesScales = []uint64{1 << 10, 1 << 20, 1 << 30, 1 << 40, 1 << 60}
	bytesNames  = []string{"KB", "MB", "GB", "TB", "PB"}
)

func bytesHuman(size uint64) string {
	for i := 1; i < len(bytesScales); i++ {
		if size < bytesScales[i] {
			return fmt.Sprintf("%.2f %s", float64(size)/float64(bytesScales[i-1]), bytesNames[i-1])
		}
	}

	return fmt.Sprintf("%d bytes", size)
}
