package registry

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"sync/atomic"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
)

type Registry[T any] struct {
	exclusive   exclusive.Service
	storage     absto.Storage
	redisClient redis.Client
	current     atomic.Pointer[map[string]T]
	filename    string
	channel     string
	lockName    string
	writeMutex  sync.Mutex
}

func New[T any](storageService absto.Storage, exclusiveService exclusive.Service, redisClient redis.Client, filename, channel string) *Registry[T] {
	return &Registry[T]{
		storage:     storageService,
		exclusive:   exclusiveService,
		redisClient: redisClient,
		filename:    filename,
		channel:     channel,
		lockName:    "fibr:mutex:" + filename,
	}
}

// Items returns the current snapshot, which is never modified once published: callers must not modify it.
func (r *Registry[T]) Items() map[string]T {
	if items := r.current.Load(); items != nil {
		return *items
	}

	return nil
}

func (r *Registry[T]) Load(ctx context.Context) error {
	r.writeMutex.Lock()
	defer r.writeMutex.Unlock()

	items, err := r.read(ctx)
	if err != nil {
		return err
	}

	r.current.Store(&items)

	return nil
}

// Update runs action on a private copy of the file content and returns the changes to save and publish.
// The cross instance lock is held from the read to the publish, so concurrent writers never overwrite each other,
// and readers only see the result once it is saved.
func (r *Registry[T]) Update(ctx context.Context, action func(items map[string]T) ([]T, error)) error {
	return r.exclusive.Execute(ctx, r.lockName, exclusive.Duration, func(ctx context.Context) error {
		r.writeMutex.Lock()
		defer r.writeMutex.Unlock()

		items, err := r.read(ctx)
		if err != nil {
			return fmt.Errorf("refresh `%s`: %w", r.filename, err)
		}

		changes, err := action(items)
		if err != nil {
			return err
		}

		if len(changes) != 0 {
			if err = provider.SaveJSON(ctx, r.storage, r.filename, items); err != nil {
				return fmt.Errorf("save `%s`: %w", r.filename, err)
			}
		}

		r.current.Store(&items)

		var errs []error

		for _, change := range changes {
			if err = r.redisClient.PublishJSON(ctx, r.channel, change); err != nil {
				errs = append(errs, fmt.Errorf("publish change of `%s`: %w", r.filename, err))
			}
		}

		return errors.Join(errs...)
	})
}

func (r *Registry[T]) Patch(apply func(items map[string]T)) {
	r.writeMutex.Lock()
	defer r.writeMutex.Unlock()

	next := maps.Clone(r.Items())
	if next == nil {
		next = make(map[string]T)
	}

	apply(next)

	r.current.Store(&next)
}

func (r *Registry[T]) read(ctx context.Context) (map[string]T, error) {
	items, err := provider.LoadJSON[map[string]T](ctx, r.storage, r.filename)
	if err == nil {
		if items == nil {
			items = make(map[string]T)
		}

		return items, nil
	}

	if !absto.IsNotExist(err) {
		return nil, err
	}

	if err = r.storage.Mkdir(ctx, provider.MetadataDirectoryName); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}

	items = make(map[string]T)

	if err = provider.SaveJSON(ctx, r.storage, r.filename, items); err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}

	return items, nil
}
