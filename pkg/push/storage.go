package push

import (
	"context"
	"errors"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
)

var (
	subscriptionsFilename = "subscriptions.json"
	ErrNotFound           = errors.New("subscription not found")
)

func (s *Service) Find(ctx context.Context, item absto.Item, url string) (Subscription, error) {
	subscriptions, err := s.get(ctx, item)
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

func (s *Service) Add(ctx context.Context, item absto.Item, subscription Subscription) error {
	return s.exclusive.Execute(ctx, "fibr:mutex:push:"+item.ID, exclusive.Duration, func(ctx context.Context) error {
		subscriptions, err := s.get(ctx, item)
		if err != nil {
			return fmt.Errorf("get subscriptions: %w", err)
		}

		for _, sub := range subscriptions {
			if sub.Auth == subscription.Auth && sub.PublicKey == subscription.PublicKey {
				return nil
			}
		}

		subscriptions = append(subscriptions, subscription)

		return s.save(ctx, item, subscriptions)
	})
}

func (s *Service) get(ctx context.Context, item absto.Item) ([]Subscription, error) {
	subscriptions, err := provider.LoadJSON[[]Subscription](ctx, s.storage, provider.MetadataDirectory(item)+subscriptionsFilename)
	if err != nil && !absto.IsNotExist(err) {
		return nil, err
	}

	return subscriptions, nil
}

func (s *Service) save(ctx context.Context, item absto.Item, subscriptions []Subscription) error {
	return provider.SaveJSON(ctx, s.storage, provider.MetadataDirectory(item)+subscriptionsFilename, subscriptions)
}
