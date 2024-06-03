package thumbnail

import (
	"context"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/version"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"github.com/ViBiOh/vith/pkg/model"
)

func (s Service) CanHaveThumbnail(item absto.Item) bool {
	return !item.IsDir() && provider.ThumbnailExtensions[item.Extension] && (s.maxSize == 0 || item.Size() < s.maxSize || s.directAccess)
}

func (s Service) CanGenerateThumbnail(item absto.Item) bool {
	return !item.IsDir() && provider.VithExtensions[item.Extension] && (s.maxSize == 0 || item.Size() < s.maxSize || s.directAccess)
}

func (s Service) HasLargeThumbnail(ctx context.Context, item absto.Item) bool {
	if s.largeSize == 0 {
		return false
	}

	return s.HasThumbnail(ctx, item, s.largeSize)
}

func (s Service) HasThumbnail(ctx context.Context, item absto.Item, scale uint64) bool {
	if item.IsDir() {
		return false
	}

	_, err := s.Info(ctx, s.PathForScale(item, scale))
	return err == nil
}

func (s Service) Path(item absto.Item) string {
	return s.PathForScale(item, SmallSize)
}

func (s Service) PathForLarge(item absto.Item) string {
	return s.PathForScale(item, s.largeSize)
}

func (s Service) PathForScale(item absto.Item, scale uint64) string {
	if item.IsDir() {
		return provider.MetadataDirectory(item)
	}

	switch scale {
	case s.largeSize:
		item.ID = item.ID + "_large"
	}

	return getThumbnailPathForExtension(item, "webp")
}

func (s Service) GetChunk(ctx context.Context, pathname string) (absto.Item, error) {
	return s.Info(ctx, provider.MetadataDirectoryName+pathname)
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
	}

	if _, ok := provider.PdfExtensions[item.Extension]; ok {
		itemType = model.TypePDF
	}

	return itemType
}

func redisKey(filename string) string {
	return version.Redis("thumbnail:" + provider.Hash(filename))
}

func (s Service) Info(ctx context.Context, pathname string) (item absto.Item, err error) {
	ctx, end := telemetry.StartSpan(ctx, s.tracer, "info")
	defer end(&err)

	return s.cache.Get(ctx, pathname)
}
