package push

import (
	"context"
	"errors"
	"fmt"
	"slices"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
)

var (
	subscriptionsFilename   = provider.MetadataDirectoryName + "/subscriptions.json"
	ErrNotFound             = errors.New("subscription not found")
	ErrTooManySubscriptions = errors.New("too many push subscriptions")
)

func (s *Service) Find(ctx context.Context, url string) (Subscription, error) {
	subscriptions, err := s.get(ctx)
	if err != nil {
		return Subscription{}, fmt.Errorf("get: %w", err)
	}

	for _, sub := range subscriptions {
		if sub.Endpoint == url {
			return sub, nil
		}
	}

	return Subscription{}, ErrNotFound
}

func (s *Service) Add(ctx context.Context, subscription Subscription) error {
	return s.exclusive.Execute(ctx, "fibr:mutex:push", exclusive.Duration, func(ctx context.Context) error {
		subscriptions, err := s.get(ctx)
		if err != nil {
			return fmt.Errorf("get subscriptions: %w", err)
		}

		for _, sub := range subscriptions {
			if sub.Similar(subscription) {
				return nil
			}
		}

		if s.maxSubscriptions > 0 && len(subscriptions) >= s.maxSubscriptions {
			return ErrTooManySubscriptions
		}

		subscriptions = append(subscriptions, subscription)

		return s.save(ctx, subscriptions)
	})
}

func (s *Service) Delete(ctx context.Context, endpoint string) error {
	return s.exclusive.Execute(ctx, "fibr:mutex:push", exclusive.Duration, func(ctx context.Context) error {
		subscriptions, err := s.get(ctx)
		if err != nil {
			return fmt.Errorf("get subscriptions: %w", err)
		}

		remaining := slices.DeleteFunc(slices.Clone(subscriptions), func(sub Subscription) bool {
			return sub.Endpoint == endpoint
		})

		if len(remaining) == len(subscriptions) {
			return nil
		}

		return s.save(ctx, remaining)
	})
}

func (s *Service) get(ctx context.Context) ([]Subscription, error) {
	subscriptions, err := provider.LoadJSON[[]Subscription](ctx, s.storage, subscriptionsFilename)
	if err != nil && !absto.IsNotExist(err) {
		return nil, err
	}

	return subscriptions, nil
}

func (s *Service) save(ctx context.Context, subscriptions []Subscription) error {
	return provider.SaveJSON(ctx, s.storage, subscriptionsFilename, subscriptions)
}
