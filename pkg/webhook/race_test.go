package webhook

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	"go.uber.org/mock/gomock"
)

func TestConcurrentMutationAndRead(t *testing.T) {
	t.Parallel()

	storage, err := filesystem.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	redisClient := mocks.NewRedisClient(gomock.NewController(t))
	redisClient.EXPECT().PublishJSON(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	instance := &Service{
		storage:          storage,
		redisClient:      redisClient,
		exclusiveService: exclusive.New(nil),
		webhooks:         make(map[string]provider.Webhook),
	}

	ctx := context.Background()

	var wg sync.WaitGroup

	for index := range 20 {
		wg.Add(3)

		go func() {
			defer wg.Done()

			id, err := instance.Create(ctx, "/website", false, provider.Raw, fmt.Sprintf("http://127.0.0.1/%d", index), []provider.EventType{provider.AccessEvent})
			if err != nil {
				t.Errorf("Create() = %s", err)
				return
			}

			if err = instance.Delete(ctx, id); err != nil {
				t.Errorf("Delete() = %s", err)
			}
		}()

		go func() {
			defer wg.Done()

			instance.matchingWebhooks(provider.Event{Type: provider.AccessEvent})
		}()

		go func() {
			defer wg.Done()

			instance.List()
		}()
	}

	wg.Wait()
}
