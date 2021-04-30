package crud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var (
	metadataFilename = path.Join(provider.MetadataDirectoryName, ".json")
)

// Clock give time
type Clock struct {
	now time.Time
}

// Now return current time
func (c *Clock) Now() time.Time {
	if c == nil {
		return time.Now()
	}
	return c.now
}

func (a *app) dumpMetadatas() map[string]provider.Share {
	metadatas := make(map[string]provider.Share, 0)

	a.metadatas.Range(func(key interface{}, value interface{}) bool {
		metadatas[key.(string)] = value.(provider.Share)
		return true
	})

	return metadatas
}

func (a *app) refreshMetadatas() error {
	_, err := a.storage.Info(metadataFilename)
	if err != nil && !provider.IsNotExist(err) {
		return err
	}

	if provider.IsNotExist(err) {
		if err := a.storage.CreateDir(provider.MetadataDirectoryName); err != nil {
			return err
		}

		return nil
	}

	file, err := a.storage.ReaderFrom(metadataFilename)
	if file != nil {
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				err = fmt.Errorf("%s: %w", err, closeErr)
			}
		}()
	}
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(file)
	var metadatas map[string]provider.Share
	if err = decoder.Decode(&metadatas); err != nil {
		return fmt.Errorf("unable to decode metadatas: %s", err)
	}

	for _, metadata := range metadatas {
		a.metadatas.Store(metadata.ID, metadata)
	}

	return nil
}

func (a *app) purgeExpiredMetadatas() bool {
	now := a.clock.Now()
	changed := false

	a.metadatas.Range(func(key interface{}, value interface{}) bool {
		share := value.(provider.Share)

		if share.IsExpired(now) {
			changed = true
			a.metadatas.Delete(key)
		}

		return true
	})

	return changed
}

func (a *app) cleanMetadatas(_ context.Context) error {
	if a.highAvailability {
		lockFilename := path.Join(provider.MetadataDirectoryName, ".lock")
		acquired, err := a.storage.Semaphore(lockFilename)
		if err != nil {
			return fmt.Errorf("unable to create lock file: %s", err)
		}

		if !acquired {
			logger.Info("metadatas purge is already in progress: lock file creation failed")
			return nil
		}

		defer func() {
			if err := a.storage.Remove(lockFilename); err != nil {
				logger.WithField("filename", lockFilename).Error("unable to remove lock file: %s", err)
			}
		}()
	}

	if a.purgeExpiredMetadatas() {
		if err := a.saveMetadatas(); err != nil {
			return fmt.Errorf("unable to save metadatas: %s", err)
		}
	}

	return nil
}

func (a *app) saveMetadatas() (err error) {
	if !a.metadataEnabled {
		return errors.New("metadata not enabled")
	}

	content, err := json.MarshalIndent(a.dumpMetadatas(), "", "  ")
	if err != nil {
		return err
	}

	file, err := a.storage.WriterTo(metadataFilename)
	if file != nil {
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				err = fmt.Errorf("%s: %w", err, closeErr)
			}
		}()
	}
	if err != nil {
		return err
	}

	n, err := file.Write(content)
	if err != nil {
		return err
	}

	if n < len(content) {
		return io.ErrShortWrite
	}

	return nil
}
