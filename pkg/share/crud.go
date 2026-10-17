package share

import (
	"context"
	"fmt"
	"path"
	"sort"
	"time"

	"codeberg.org/ViBiOh/auth/v3/pkg/argon"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func generateID(shares map[string]provider.Share) string {
	for {
		idSha := provider.Hash(provider.Identifier())[:8]

		if _, ok := shares[idSha]; !ok {
			return idSha
		}
	}
}

func (s *Service) List() []provider.Share {
	shares := s.shares.Items()

	output := make([]provider.Share, 0, len(shares))

	for _, value := range shares {
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

	err := s.shares.Update(ctx, func(shares map[string]provider.Share) ([]provider.Share, error) {
		id = generateID(shares)

		share := provider.Share{
			ID:       id,
			Path:     filepath,
			RootName: path.Base(filepath),
			Edit:     edit,
			Story:    story,
			Password: password,
			File:     !isDir,
			Created:  s.clock(),
			Duration: duration,
		}

		shares[id] = share

		return []provider.Share{share}, nil
	})

	return id, err
}

func (s *Service) UpdatePassword(ctx context.Context, id, password string) error {
	hashed, err := argon.GenerateFromPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return s.shares.Update(ctx, func(shares map[string]provider.Share) ([]provider.Share, error) {
		share, ok := shares[id]
		if !ok {
			return nil, fmt.Errorf("unknown share `%s`", id)
		}

		share.Password = hashed
		shares[id] = share

		return []provider.Share{share}, nil
	})
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.shares.Update(ctx, func(shares map[string]provider.Share) ([]provider.Share, error) {
		delete(shares, id)

		return []provider.Share{{ID: id}}, nil
	})
}
