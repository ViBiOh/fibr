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
	Get([]byte) (map[string]interface{}, error)
	Rename([]byte, []byte) error
	HasEntry([]byte) bool
	Store([]byte, []byte) error
	Delete([]byte) error
	Close() error
}

type app struct {
	db *badger.DB
}

// New creates new App
func New(storageApp provider.Storage) (App, error) {
	db, err := badger.Open(badger.DefaultOptions(storageApp.Path(path.Join(provider.MetadataDirectoryName, "LmZpYnI="))).WithMemTableSize(1024 * 1024 * 8).WithNumMemtables(1).WithNumLevelZeroTables(1).WithNumLevelZeroTablesStall(2))
	if err != nil {
		return nil, fmt.Errorf("unable to open database: %s", err)
	}

	return app{
		db: db,
	}, nil
}

func (a app) Get(key []byte) (map[string]interface{}, error) {
	var content map[string]interface{}

	err := a.db.View(func(txn *badger.Txn) error {
		entry, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}

			return fmt.Errorf("unable to get entry: %s", err)
		}

		if err := entry.Value(func(val []byte) error {
			return json.Unmarshal(val, &content)
		}); err != nil {
			return fmt.Errorf("unable to read entry: %s", err)
		}

		return nil
	})

	return content, err
}

func (a app) Rename(old, new []byte) error {
	return a.db.Update(func(txn *badger.Txn) error {
		entry, err := txn.Get(old)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}

			return fmt.Errorf("unable to get entry: %s", err)
		}

		if err := txn.Delete(old); err != nil {
			return fmt.Errorf("unable to delete entry: %s", err)
		}

		if err := entry.Value(func(content []byte) error {
			if err := txn.Set(new, content); err != nil {
				return fmt.Errorf("unable to set entry: %s", err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("unable to rename entry: %s", err)
		}

		return nil
	})
}
func (a app) HasEntry(key []byte) bool {
	return a.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	}) != badger.ErrKeyNotFound
}

func (a app) Delete(key []byte) error {
	return a.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (a app) Store(key, value []byte) error {
	return a.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

func (a app) Close() error {
	return a.db.Close()
}
