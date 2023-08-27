package exclusive

import (
	"context"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/redis"
)

const Duration = time.Second * 10

type Service struct {
	redisClient redis.Client
}

func New(redisClient redis.Client) Service {
	return Service{
		redisClient: redisClient,
	}
}

func (s Service) Enabled() bool {
	return s.redisClient != nil && s.redisClient.Enabled()
}

func (s Service) Execute(ctx context.Context, name string, duration time.Duration, action func(context.Context) error) error {
	if !s.Enabled() {
		return action(ctx)
	}

exclusive:
	acquired, err := s.redisClient.Exclusive(ctx, name, duration, action)
	if err != nil {
		return err
	}

	if !acquired {
		time.Sleep(time.Second)
		goto exclusive
	}

	return nil
}

func (s Service) Try(ctx context.Context, name string, duration time.Duration, action func(context.Context) error) (bool, error) {
	if !s.Enabled() {
		return true, action(ctx)
	}

	return s.redisClient.Exclusive(ctx, name, duration, action)
}
