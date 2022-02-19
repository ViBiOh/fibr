package share

import (
	"context"
	"fmt"
	"path"
	"sort"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/httputils/v4/pkg/uuid"
)

func (a *App) generateID() (string, error) {
	for {
		uuid, err := uuid.New()
		if err != nil {
			return "", err
		}
		id := sha.New(uuid)[:8]

		if _, ok := a.shares[id]; !ok {
			return id, nil
		}
	}
}

// List shares
func (a *App) List() (shares []provider.Share) {
	a.RLock()

	var i int64
	shares = make([]provider.Share, len(a.shares))

	for _, value := range a.shares {
		shares[i] = value
		i++
	}

	a.RUnlock()

	sort.Sort(provider.ShareByID(shares))

	return shares
}

// Create a share
func (a *App) Create(ctx context.Context, filepath string, edit bool, password string, isDir bool, duration time.Duration) (string, error) {
	var id string

	_, err := a.Exclusive(ctx, a.amqpExclusiveRoutingKey, semaphoreDuration, func(_ context.Context) error {
		var err error
		id, err = a.generateID()
		if err != nil {
			return fmt.Errorf("unable to generate id: %s", err)
		}

		share := provider.Share{
			ID:       id,
			Path:     filepath,
			RootName: path.Base(filepath),
			Edit:     edit,
			Password: password,
			File:     !isDir,
			Creation: a.clock.Now(),
			Duration: duration,
		}

		a.shares[id] = share

		if err = provider.SaveJSON(a.storageApp, shareFilename, a.shares); err != nil {
			return fmt.Errorf("unable to save shares: %s", err)
		}

		if a.amqpClient != nil {
			if err = a.amqpClient.PublishJSON(share, a.amqpExchange, a.amqpRoutingKey); err != nil {
				return fmt.Errorf("unable to publish share creation: %s", err)
			}
		}

		return nil
	})

	return id, err
}

// Delete a share
func (a *App) Delete(ctx context.Context, id string) error {
	_, err := a.Exclusive(ctx, a.amqpExclusiveRoutingKey, semaphoreDuration, func(_ context.Context) error {
		return a.delete(id)
	})

	return err
}

func (a *App) delete(id string) error {
	delete(a.shares, id)

	if err := provider.SaveJSON(a.storageApp, shareFilename, a.shares); err != nil {
		return fmt.Errorf("unable to save shares: %s", err)
	}

	if a.amqpClient != nil {
		if err := a.amqpClient.PublishJSON(provider.Share{ID: id}, a.amqpExchange, a.amqpRoutingKey); err != nil {
			return fmt.Errorf("unable to publish share deletion: %s", err)
		}
	}

	return nil
}
