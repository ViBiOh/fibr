package thumbnail

import (
	"context"

	absto "github.com/ViBiOh/absto/pkg/model"
)

func (s Service) ListDirLarge(ctx context.Context, item absto.Item) (map[string]absto.Item, error) {
	return s.listDirectoryForScale(ctx, item, s.largeStorage)
}

func (s Service) ListDir(ctx context.Context, item absto.Item) (map[string]absto.Item, error) {
	return s.listDirectoryForScale(ctx, item, s.smallStorage)
}

func (s Service) listDirectoryForScale(ctx context.Context, item absto.Item, storageService absto.Storage) (map[string]absto.Item, error) {
	if !item.IsDir() {
		return nil, nil
	}

	list, err := storageService.List(ctx, s.Path(item))
	if err != nil && !absto.IsNotExist(err) {
		return nil, err
	}

	thumbnails := make(map[string]absto.Item, len(list))
	for _, thumbnail := range list {
		thumbnails[thumbnail.Pathname] = thumbnail
	}

	return thumbnails, nil
}
