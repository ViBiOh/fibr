package crud

import (
	"context"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
)

type cover struct {
	Img       provider.RenderItem
	ImgHeight uint64
	ImgWidth  uint64
}

func newCover(item provider.RenderItem, size uint64) cover {
	return cover{
		Img:       item,
		ImgHeight: size,
		ImgWidth:  size,
	}
}

func (c cover) IsZero() bool {
	return len(c.Img.Item.Name) == 0
}

func (a App) getCover(ctx context.Context, request provider.Request, files []absto.Item) (output cover) {
	for _, file := range files {
		if a.thumbnailApp.HasThumbnail(ctx, file, thumbnail.SmallSize) {
			output = newCover(provider.StorageToRender(file, request), thumbnail.SmallSize)
			return
		}
	}

	return
}
