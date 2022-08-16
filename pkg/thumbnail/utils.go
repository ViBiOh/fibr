package thumbnail

import (
	"context"
	"fmt"

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
	_, ok := a.ThumbnailInfo(ctx, item, scale)
	return ok
}

// ThumbnailInfo determine if thumbnail exist for given pathname and provide detail about it
func (a App) ThumbnailInfo(ctx context.Context, item absto.Item, scale uint64) (thumbnailItem absto.Item, ok bool) {
	if item.IsDir {
		ok = false
		return
	}

	var err error
	thumbnailItem, err = a.storageApp.Info(ctx, a.PathForScale(item, scale))
	ok = err == nil
	return
}

// Path computes thumbnail path for a a given item
func (a App) Path(item absto.Item) string {
	return a.PathForScale(item, SmallSize)
}

// PathForLarge computes thumbnail path for a a given item and large size
func (a App) PathForLarge(item absto.Item) string {
	return a.PathForScale(item, a.largeSize)
}

// PathForScale computes thumbnail path for a a given item and scale
func (a App) PathForScale(item absto.Item, scale uint64) string {
	if item.IsDir {
		return provider.MetadataDirectory(item)
	}

	switch scale {
	case a.largeSize:
		item.ID = item.ID + "_large"
	}

	return getThumbnailPathForExtension(item, "webp")
}

// GetChunk retrieve the storage item in the metadata
func (a App) GetChunk(ctx context.Context, pathname string) (absto.Item, error) {
	return a.storageApp.Info(ctx, provider.MetadataDirectoryName+pathname)
}

func getStreamPath(item absto.Item) string {
	return getThumbnailPathForExtension(item, "m3u8")
}

// PathWithExtension computes thumbnail path with given extension
func getThumbnailPathForExtension(item absto.Item, extension string) string {
	return fmt.Sprintf("%s%s.%s", provider.MetadataDirectory(item), item.ID, extension)
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
