package thumbnail

import (
	"context"
	"fmt"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/version"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/ViBiOh/vith/pkg/model"
)

var redisCacheDuration = time.Hour * 96

func (a App) CanHaveThumbnail(item absto.Item) bool {
	return !item.IsDir && provider.ThumbnailExtensions[item.Extension] && (a.maxSize == 0 || item.Size < a.maxSize || a.directAccess)
}

func (a App) HasLargeThumbnail(ctx context.Context, item absto.Item) bool {
	if a.largeSize == 0 {
		return false
	}

	return a.HasThumbnail(ctx, item, a.largeSize)
}

func (a App) HasThumbnail(ctx context.Context, item absto.Item, scale uint64) bool {
	if item.IsDir {
		return false
	}

	_, err := a.Info(ctx, a.PathForScale(item, scale))
	return err == nil
}

func (a App) Path(item absto.Item) string {
	return a.PathForScale(item, SmallSize)
}

func (a App) PathForLarge(item absto.Item) string {
	return a.PathForScale(item, a.largeSize)
}

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

func (a App) GetChunk(ctx context.Context, pathname string) (absto.Item, error) {
	return a.Info(ctx, provider.MetadataDirectoryName+pathname)
}

func getStreamPath(item absto.Item) string {
	return getThumbnailPathForExtension(item, "m3u8")
}

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

func redisKey(filename string) string {
	return version.Redis("thumbnail:" + sha.New(filename))
}

func (a App) Info(ctx context.Context, pathname string) (absto.Item, error) {
	ctx, end := tracer.StartSpan(ctx, a.tracer, "info")
	defer end()

	return a.cacheApp.Get(ctx, pathname)
}
