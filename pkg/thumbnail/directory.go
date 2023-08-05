package thumbnail

import (
	"context"

	absto "github.com/ViBiOh/absto/pkg/model"
)

func (a App) ListDirLarge(ctx context.Context, item absto.Item) (map[string]absto.Item, error) {
	return a.listDirectoryForScale(ctx, item, a.largeStorageApp)
}

func (a App) ListDir(ctx context.Context, item absto.Item) (map[string]absto.Item, error) {
	return a.listDirectoryForScale(ctx, item, a.smallStorageApp)
}

func (a App) listDirectoryForScale(ctx context.Context, item absto.Item, storageApp absto.Storage) (map[string]absto.Item, error) {
	if !item.IsDir() {
		return nil, nil
	}

	list, err := storageApp.List(ctx, a.Path(item))
	if err != nil && !absto.IsNotExist(err) {
		return nil, err
	}

	thumbnails := make(map[string]absto.Item, len(list))
	for _, thumbnail := range list {
		thumbnails[thumbnail.Pathname] = thumbnail
	}

	return thumbnails, nil
}
