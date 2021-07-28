package database

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	badger "github.com/dgraph-io/badger/v3"
)

// App of package
type App interface {
	Get(provider.StorageItem) (map[string]interface{}, error)
	HasEntry(provider.StorageItem) bool
	Store([]byte, []byte) error
	Close() error
}

type app struct {
	db *badger.DB
}

// New creates new App
func New(storageApp provider.Storage) (App, error) {
	db, err := badger.Open(badger.DefaultOptions(storageApp.Path(path.Join(provider.MetadataDirectoryName, "database"))))
	if err != nil {
		return nil, fmt.Errorf("unable to open database: %s", err)
	}

	return app{
		db: db,
	}, nil
}

func (a app) Get(item provider.StorageItem) (map[string]interface{}, error) {
	var content map[string]interface{}

	err := a.db.View(func(txn *badger.Txn) error {
		entry, err := txn.Get([]byte(item.Pathname))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}

			return fmt.Errorf("unable to get entry for `%s`: %s", item.Pathname, err)
		}

		if err := entry.Value(func(val []byte) error {
			return json.Unmarshal(val, &content)
		}); err != nil {
			return fmt.Errorf("unable to read entry for `%s`: %s", item.Pathname, err)
		}

		return nil
	})

	return content, err
}

func (a app) HasEntry(item provider.StorageItem) bool {
	return a.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(item.Pathname))
		return err
	}) != badger.ErrKeyNotFound
}

func (a app) Store(key, value []byte) error {
	return a.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func (a app) Close() error {
	return a.db.Close()
}
