package search

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exif"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	amqpExclusiveRoutingKey *string
}

func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		amqpExclusiveRoutingKey: flags.String(fs, prefix, "search", "AmqpExclusiveRoutingKey", "AMQP Routing Key for exclusive lock on default exchange", "fibr.semaphore.search", nil),
	}
}

type App struct {
	tracer                  trace.Tracer
	storageApp              absto.Storage
	exifApp                 provider.ExifManager
	amqpClient              *amqp.Client
	amqpExclusiveRoutingKey string
	thumbnailApp            thumbnail.App
}

func New(config Config, storageApp absto.Storage, thumbnailApp thumbnail.App, exifApp exif.App, amqpClient *amqp.Client, tracer trace.Tracer) (App, error) {
	var amqpExclusiveRoutingKey string

	if amqpClient != nil {
		amqpExclusiveRoutingKey = strings.TrimSpace(*config.amqpExclusiveRoutingKey)

		if err := amqpClient.SetupExclusive(amqpExclusiveRoutingKey); err != nil {
			return App{}, fmt.Errorf("setup amqp exclusive: %w", err)
		}
	}

	return App{
		tracer:       tracer,
		storageApp:   storageApp,
		thumbnailApp: thumbnailApp,
		exifApp:      exifApp,

		amqpExclusiveRoutingKey: amqpExclusiveRoutingKey,
	}, nil
}

func (a App) Exclusive(ctx context.Context, name string, duration time.Duration, action func(ctx context.Context) error) error {
	if a.amqpClient == nil {
		return action(ctx)
	}

exclusive:
	acquired, err := a.amqpClient.Exclusive(ctx, name, duration, func(ctx context.Context) error {
		return action(ctx)
	})
	if err != nil {
		return err
	}

	if !acquired {
		time.Sleep(time.Second)
		goto exclusive
	}

	return nil
}

func (a App) Files(r *http.Request, request provider.Request) (items []absto.Item, err error) {
	params := r.URL.Query()

	criterions, err := parseSearch(params, time.Now())
	if err != nil {
		return nil, httpModel.WrapInvalid(err)
	}

	err = a.storageApp.Walk(r.Context(), request.Filepath(), func(item absto.Item) error {
		if item.IsDir || !criterions.match(item) {
			return nil
		}

		items = append(items, item)

		return nil
	})

	return
}

func (a App) Search(r *http.Request, request provider.Request, files []absto.Item) ([]provider.RenderItem, bool, error) {
	ctx, end := tracer.StartSpan(r.Context(), a.tracer, "search")
	defer end()

	items := make([]provider.RenderItem, len(files))
	var hasMap bool

	renderWithThumbnail := request.Display == provider.GridDisplay

	for i, item := range files {
		renderItem := provider.StorageToRender(item, request)

		if renderWithThumbnail && a.thumbnailApp.CanHaveThumbnail(item) && a.thumbnailApp.HasThumbnail(ctx, item, thumbnail.SmallSize) {
			renderItem.HasThumbnail = true
		}

		items[i] = renderItem

		if !hasMap {
			if exif, err := a.exifApp.GetExifFor(ctx, item); err == nil && exif.Geocode.Longitude != 0 && exif.Geocode.Latitude != 0 {
				hasMap = true
			}
		}
	}

	return items, hasMap, nil
}
