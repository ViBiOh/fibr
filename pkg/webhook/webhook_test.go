package webhook

import (
	"sync"
	"testing"
	"time"

	"github.com/ViBiOh/absto/pkg/filesystem"
	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type publishedMessages struct {
	messages []provider.Webhook
	mutex    sync.Mutex
}

func (p *publishedMessages) all() []provider.Webhook {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return append([]provider.Webhook(nil), p.messages...)
}

func newTestService(t *testing.T, webhooks ...provider.Webhook) (*Service, absto.Storage, *publishedMessages) {
	t.Helper()

	storage, err := filesystem.New(t.TempDir())
	require.NoError(t, err)

	published := &publishedMessages{}

	redisClient := mocks.NewRedisClient(gomock.NewController(t))
	redisClient.EXPECT().PublishJSON(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, _ string, value any) error {
		published.mutex.Lock()
		defer published.mutex.Unlock()

		published.messages = append(published.messages, value.(provider.Webhook))

		return nil
	}).AnyTimes()

	instance := &Service{
		redisClient:   redisClient,
		pubsubChannel: "channel",
		webhooks:      registry.New[provider.Webhook](storage, exclusive.New(nil), redisClient, webhookFilename, "channel"),
	}

	if len(webhooks) != 0 {
		seed := make(map[string]provider.Webhook, len(webhooks))
		for _, webhook := range webhooks {
			seed[webhook.ID] = webhook
		}

		require.NoError(t, storage.Mkdir(t.Context(), provider.MetadataDirectoryName))
		require.NoError(t, provider.SaveJSON(t.Context(), storage, webhookFilename, seed))
	}

	require.NoError(t, instance.webhooks.Load(t.Context()))

	return instance, storage, published
}

func loadWebhooks(t *testing.T, storage absto.Storage) map[string]provider.Webhook {
	t.Helper()

	webhooks, err := provider.LoadJSON[map[string]provider.Webhook](t.Context(), storage, webhookFilename)
	require.NoError(t, err)

	return webhooks
}

var createdAt = time.Date(2021, 5, 1, 12, 0, 0, 0, time.UTC)

func TestCreate(t *testing.T) {
	t.Parallel()

	instance, storage, published := newTestService(t)

	id, err := instance.Create(t.Context(), "/website/", true, provider.Raw, "https://example.com/hook", []provider.EventType{provider.UploadEvent})
	require.NoError(t, err)
	assert.Len(t, id, 8)

	created := instance.Get(id)

	assert.Equal(t, id, created.ID)
	assert.Equal(t, "/website/", created.Pathname)
	assert.Equal(t, "https://example.com/hook", created.URL)
	assert.True(t, created.Recursive)
	assert.Equal(t, []provider.Webhook{created}, instance.List())
	assert.Equal(t, []provider.Webhook{created}, instance.FindByURL("https://example.com/hook"))
	assert.Empty(t, instance.FindByURL("https://example.com/other"))

	assert.Equal(t, []string{id}, mapKeys(loadWebhooks(t, storage)))
	assert.Equal(t, []provider.Webhook{created}, published.all())
}

func TestCreateSimilar(t *testing.T) {
	t.Parallel()

	instance, storage, published := newTestService(t)

	types := []provider.EventType{provider.UploadEvent}

	first, err := instance.Create(t.Context(), "/website/", false, provider.Push, "https://fcm.googleapis.com/x", types)
	require.NoError(t, err)

	second, err := instance.Create(t.Context(), "/website/", false, provider.Push, "https://fcm.googleapis.com/x", types)
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Len(t, loadWebhooks(t, storage), 1)
	assert.Len(t, published.all(), 1)

	other, err := instance.Create(t.Context(), "/website/", false, provider.Push, "https://fcm.googleapis.com/y", types)
	require.NoError(t, err)

	assert.NotEqual(t, first, other)
	assert.Len(t, loadWebhooks(t, storage), 2)
}

func TestDelete(t *testing.T) {
	t.Parallel()

	keep := provider.Webhook{ID: "keep1234", Pathname: "/a/", URL: "https://example.com/a", Created: createdAt}
	remove := provider.Webhook{ID: "gone5678", Pathname: "/b/", URL: "https://example.com/b", Created: createdAt}

	instance, storage, published := newTestService(t, keep, remove)

	require.NoError(t, instance.Delete(t.Context(), remove.ID))

	assert.Equal(t, provider.Webhook{}, instance.Get(remove.ID))
	assert.Equal(t, keep, instance.Get(keep.ID))
	assert.Equal(t, map[string]provider.Webhook{keep.ID: keep}, loadWebhooks(t, storage))
	assert.Equal(t, []provider.Webhook{{ID: remove.ID}}, published.all())
}

func TestDeleteItem(t *testing.T) {
	t.Parallel()

	first := provider.Webhook{ID: "first123", Pathname: "/photos/", URL: "https://example.com/a", Created: createdAt}
	second := provider.Webhook{ID: "second12", Pathname: "/photos/", URL: "https://example.com/b", Created: createdAt}
	other := provider.Webhook{ID: "other123", Pathname: "/videos/", URL: "https://example.com/c", Created: createdAt}

	instance, storage, published := newTestService(t, first, second, other)

	require.NoError(t, instance.deleteItem(t.Context(), absto.Item{Pathname: "/photos/"}))

	assert.Equal(t, map[string]provider.Webhook{other.ID: other}, loadWebhooks(t, storage))
	assert.ElementsMatch(t, []provider.Webhook{{ID: first.ID}, {ID: second.ID}}, published.all())

	require.NoError(t, instance.deleteItem(t.Context(), absto.Item{Pathname: "/nothing/"}))
	assert.Len(t, published.all(), 2)
}

func TestMatchingWebhooks(t *testing.T) {
	t.Parallel()

	upload := provider.Webhook{ID: "upload12", Pathname: "/photos/", URL: "https://example.com/a", Types: []provider.EventType{provider.UploadEvent}, Created: createdAt}
	access := provider.Webhook{ID: "access12", Pathname: "/photos/", URL: "https://example.com/b", Types: []provider.EventType{provider.AccessEvent}, Created: createdAt}

	instance, _, _ := newTestService(t, upload, access)

	event := provider.Event{Type: provider.UploadEvent, Item: absto.Item{Pathname: "/photos/a.jpg", NameValue: "a.jpg"}}

	assert.Equal(t, []provider.Webhook{upload}, instance.matchingWebhooks(event))
}

func TestPubSubHandle(t *testing.T) {
	t.Parallel()

	existing := provider.Webhook{ID: "abcd1234", Pathname: "/photos/", URL: "https://example.com/a", Created: createdAt}
	added := provider.Webhook{ID: "efgh5678", Pathname: "/videos/", URL: "https://example.com/b", Created: createdAt}

	instance, _, _ := newTestService(t, existing)

	instance.PubSubHandle(added, nil)
	assert.Equal(t, added, instance.Get(added.ID))
	assert.Equal(t, existing, instance.Get(existing.ID))

	instance.PubSubHandle(provider.Webhook{ID: existing.ID}, nil)
	assert.Equal(t, provider.Webhook{}, instance.Get(existing.ID))
	assert.Equal(t, added, instance.Get(added.ID))

	instance.PubSubHandle(provider.Webhook{}, assert.AnError)
	assert.Equal(t, added, instance.Get(added.ID))
}

func mapKeys(webhooks map[string]provider.Webhook) []string {
	keys := make([]string, 0, len(webhooks))

	for key := range webhooks {
		keys = append(keys, key)
	}

	return keys
}
