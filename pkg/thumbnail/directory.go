package thumbnail

import (
	"context"

	absto "github.com/ViBiOh/absto/pkg/model"
)

// ListDirLarge return all thumbnails for large size for a given directory
func (a App) ListDirLarge(ctx context.Context, item absto.Item) ([]absto.Item, error) {
	return a.listDirectoryForScale(ctx, item, a.largeStorageApp)
}

// ListDir return all thumbnails for a given directory
func (a App) ListDir(ctx context.Context, item absto.Item) ([]absto.Item, error) {
	return a.listDirectoryForScale(ctx, item, a.smallStorageApp)
}

func (a App) listDirectoryForScale(ctx context.Context, item absto.Item, storageApp absto.Storage) ([]absto.Item, error) {
	if !item.IsDir {
		return nil, nil
	}

	thumbnails, err := storageApp.List(ctx, a.Path(item))
	if err != nil && !absto.IsNotExist(err) {
		return thumbnails, err
	}
	return thumbnails, nil
}
