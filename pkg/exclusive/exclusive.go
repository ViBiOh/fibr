package exclusive

import (
	"context"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/redis"
)

const SemaphoreDuration = time.Second * 10

type App struct {
	redisClient redis.App
}

func New(redisClient redis.App) App {
	return App{
		redisClient: redisClient,
	}
}

func (a App) Execute(ctx context.Context, name string, action func(context.Context) error) error {
	if !a.redisClient.Enabled() {
		return action(ctx)
	}

exclusive:
	acquired, err := a.redisClient.Exclusive(ctx, name, SemaphoreDuration, action)
	if err != nil {
		return err
	}

	if !acquired {
		time.Sleep(time.Second)
		goto exclusive
	}

	return nil
}

func (a App) Try(ctx context.Context, name string, action func(context.Context) error) (bool, error) {
	if !a.redisClient.Enabled() {
		return true, action(ctx)
	}

	return a.redisClient.Exclusive(ctx, name, SemaphoreDuration, action)
}
