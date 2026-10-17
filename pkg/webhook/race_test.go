package webhook

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func TestConcurrentMutationAndRead(t *testing.T) {
	t.Parallel()

	instance, _, _ := newTestService(t)

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
