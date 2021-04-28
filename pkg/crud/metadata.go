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

func (a *app) refreshMetadatas() error {
	_, err := a.storage.Info(metadataFilename)
	if err != nil && !provider.IsNotExist(err) {
		return err
	}

	if provider.IsNotExist(err) {
		if err := a.storage.CreateDir(provider.MetadataDirectoryName); err != nil {
			return err
		}

		a.metadatas = make([]*provider.Share, 0)

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
	if err = decoder.Decode(&a.metadatas); err != nil {
		return err
	}

	now := a.clock.Now()
	for _, metadata := range a.metadatas {
		if metadata.Creation.IsZero() {
			metadata.Creation = now
		}
	}

	return nil
}

func (a *app) purgeExpiredMetadatas() {
	now := a.clock.Now()

	count := 0
	for _, metadata := range a.metadatas {
		if metadata.Duration == 0 || metadata.Creation.Add(metadata.Duration).After(now) {
			a.metadatas[count] = metadata
			count++
		}
	}

	a.metadatas = a.metadatas[:count]
}

func (a *app) cleanMetadatas(_ context.Context) error {
	a.metadataLock.Lock()
	defer a.metadataLock.Unlock()

	if err := a.refreshMetadatas(); err != nil {
		return fmt.Errorf("unable to refresh metadatas: %s", err)
	}

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

	a.purgeExpiredMetadatas()

	if err := a.saveMetadata(); err != nil {
		return fmt.Errorf("unable to save metadatas: %s", err)
	}

	return nil
}

func (a *app) saveMetadata() (err error) {
	if !a.metadataEnabled {
		return errors.New("metadata not enabled")
	}

	content, err := json.MarshalIndent(&a.metadatas, "", "  ")
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
