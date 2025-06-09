package crud

import (
	"context"
	"fmt"
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

type entry struct {
	Value any
	Key   string
}

func (s *Service) stats(r *http.Request, request provider.Request, message renderer.Message) (renderer.Page, error) {
	ctx := r.Context()
	pathname := request.Filepath()

	stats, err := s.computeStats(ctx, pathname)
	if err != nil {
		return renderer.NewPage("", http.StatusInternalServerError, nil), err
	}

	entries := []entry{
		{Key: "Directories", Value: stats["Directories"]},
		{Key: "Files", Value: stats["Files"]},
		{Key: "Size", Value: bytesHuman(stats["Size"])},
		{Key: "Metadatas", Value: fmt.Sprintf("%s (%.1f%% of Size)", bytesHuman(stats["Metadatas"]), float64(stats["Metadatas"]*100)/float64(stats["Size"]))},
	}

	if request.Share.IsZero() {
		entries = append(entries, entry{Key: "Current path", Value: pathname})
	}

	return renderer.NewPage("stats", http.StatusOK, map[string]any{
		"Paths":    getPathParts(request),
		"Request":  request,
		"Message":  message,
		"Stats":    entries,
		"VapidKey": s.pushService.GetPublicKey(),
	}), nil
}

func (s *Service) computeStats(ctx context.Context, pathname string) (map[string]uint64, error) {
	var filesCount, directoriesCount, filesSize, metadataSize uint64

	err := s.storage.Walk(ctx, pathname, func(item absto.Item) error {
		if item.IsDir() {
			directoriesCount++
		} else {
			filesCount++
			filesSize += uint64(item.Size())
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("browse files: %w", err)
	}

	err = s.rawStorage.Walk(ctx, provider.MetadataDirectoryName+pathname, func(item absto.Item) error {
		if !item.IsDir() {
			metadataSize += uint64(item.Size())
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("browse metadatas: %w", err)
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
