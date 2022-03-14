package thumbnail

import (
	"context"
	"fmt"
	"path/filepath"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/vith/pkg/model"
)

// CanHaveThumbnail determine if thumbnail can be generated for given pathname
func (a App) CanHaveThumbnail(item absto.Item) bool {
	return !item.IsDir && provider.ThumbnailExtensions[item.Extension] && (a.maxSize == 0 || item.Size < a.maxSize || a.directAccess)
}

// HasLargeThumbnail determine if large thumbnail exist for given pathname
func (a App) HasLargeThumbnail(ctx context.Context, item absto.Item) bool {
	if a.largeSize == 0 {
		return false
	}

	return a.HasThumbnail(ctx, item, a.largeSize)
}

// HasThumbnail determine if thumbnail exist for given pathname
func (a App) HasThumbnail(ctx context.Context, item absto.Item, scale uint64) bool {
	if item.IsDir {
		return false
	}

	_, err := a.storageApp.Info(ctx, a.getThumbnailPath(item, scale))
	return err == nil
}

// ThumbnailInfo determine if thumbnail exist for given pathname and provide detail about it
func (a App) ThumbnailInfo(ctx context.Context, item absto.Item, scale uint64) (thumbnailItem absto.Item, ok bool) {
	if item.IsDir {
		ok = false
		return
	}

	var err error
	thumbnailItem, err = a.storageApp.Info(ctx, a.getThumbnailPath(item, scale))
	ok = err == nil
	return
}

// GetChunk retrieve the storage item in the metadata
func (a App) GetChunk(ctx context.Context, pathname string) (absto.Item, error) {
	return a.storageApp.Info(ctx, provider.MetadataDirectoryName+pathname)
}

func (a App) getThumbnailPath(item absto.Item, scale uint64) string {
	switch scale {
	case a.largeSize:
		item.ID = item.ID + "_large"
	}

	return getPathWithExtension(item, "webp")
}

func getStreamPath(item absto.Item) string {
	return getPathWithExtension(item, "m3u8")
}

func getPathWithExtension(item absto.Item, extension string) string {
	return fmt.Sprintf("%s/%s.%s", filepath.Dir(provider.MetadataDirectoryName+item.Pathname), item.ID, extension)
}

func typeOfItem(item absto.Item) model.ItemType {
	itemType := model.TypeVideo
	if _, ok := provider.ImageExtensions[item.Extension]; ok {
		itemType = model.TypeImage
	} else if _, ok := provider.PdfExtensions[item.Extension]; ok {
		itemType = model.TypePDF
	}

	return itemType
}
