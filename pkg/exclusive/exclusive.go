package exclusive

import (
	"context"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/redis"
)

const Duration = time.Second * 10

type App struct {
	redisClient redis.Client
}

func New(redisClient redis.Client) App {
	return App{
		redisClient: redisClient,
	}
}

func (a App) Enabled() bool {
	return a.redisClient != nil && a.redisClient.Enabled()
}

func (a App) Execute(ctx context.Context, name string, duration time.Duration, action func(context.Context) error) error {
	if !a.Enabled() {
		return action(ctx)
	}

exclusive:
	acquired, err := a.redisClient.Exclusive(ctx, name, duration, action)
	if err != nil {
		return err
	}

	if !acquired {
		time.Sleep(time.Second)
		goto exclusive
	}

	return nil
}

func (a App) Try(ctx context.Context, name string, duration time.Duration, action func(context.Context) error) (bool, error) {
	if !a.Enabled() {
		return true, action(ctx)
	}

	return a.redisClient.Exclusive(ctx, name, duration, action)
}
