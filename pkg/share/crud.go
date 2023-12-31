package share

import (
	"context"
	"fmt"
	"path"
	"sort"
	"time"

	"github.com/ViBiOh/auth/v2/pkg/argon"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func (s *Service) generateID() (string, error) {
	for {
		idSha := provider.Hash(provider.Identifier())[:8]

		if _, ok := s.shares[idSha]; !ok {
			return idSha, nil
		}
	}
}

func (s *Service) List() (output []provider.Share) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	output = make([]provider.Share, 0, len(s.shares))

	for _, value := range s.shares {
		index := sort.Search(len(output), func(i int) bool {
			return output[i].ID > value.ID
		})

		output = append(output, value)
		copy(output[index+1:], output[index:])
		output[index] = value
	}

	return output
}

func (s *Service) Create(ctx context.Context, filepath string, edit, story bool, password string, isDir bool, duration time.Duration) (string, error) {
	var id string

	_, err := s.Exclusive(ctx, "create", exclusive.Duration, func(ctx context.Context) error {
		var err error
		id, err = s.generateID()
		if err != nil {
			return fmt.Errorf("generate id: %w", err)
		}

		share := provider.Share{
			ID:       id,
			Path:     filepath,
			RootName: path.Base(filepath),
			Edit:     edit,
			Story:    story,
			Password: password,
			File:     !isDir,
			Creation: s.clock(),
			Duration: duration,
		}

		s.shares[id] = share

		if err = provider.SaveJSON(ctx, s.storage, shareFilename, s.shares); err != nil {
			return fmt.Errorf("save shares: %w", err)
		}

		if err = s.redisClient.PublishJSON(ctx, s.pubsubChannel, share); err != nil {
			return fmt.Errorf("publish share creation: %w", err)
		}

		return nil
	})

	return id, err
}

func (s *Service) UpdatePassword(ctx context.Context, id, password string) error {
	_, err := s.Exclusive(ctx, id, exclusive.Duration, func(_ context.Context) error {
		hashed, err := argon.GenerateFromPassword(password)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}

		share := s.shares[id]
		share.Password = hashed
		s.shares[id] = share

		if err := provider.SaveJSON(ctx, s.storage, shareFilename, s.shares); err != nil {
			return fmt.Errorf("save shares: %w", err)
		}

		if err := s.redisClient.PublishJSON(ctx, s.pubsubChannel, provider.Share{ID: id}); err != nil {
			return fmt.Errorf("publish share deletion: %w", err)
		}

		return nil
	})

	return err
}

func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.Exclusive(ctx, id, exclusive.Duration, func(_ context.Context) error {
		return s.delete(ctx, id)
	})

	return err
}

func (s *Service) delete(ctx context.Context, id string) error {
	delete(s.shares, id)

	if err := provider.SaveJSON(ctx, s.storage, shareFilename, s.shares); err != nil {
		return fmt.Errorf("save shares: %w", err)
	}

	if err := s.redisClient.PublishJSON(ctx, s.pubsubChannel, provider.Share{ID: id}); err != nil {
		return fmt.Errorf("publish share deletion: %w", err)
	}

	return nil
}
