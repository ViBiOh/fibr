package metadata

import (
	"crypto/rand"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/sha"
)

func uuid() (string, error) {
	raw := make([]byte, 16)
	_, _ = rand.Read(raw)

	raw[8] = raw[8]&^0xc0 | 0x80
	raw[6] = raw[6]&^0xf0 | 0x40

	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:]), nil
}

func (a *app) generateShareID() (string, error) {
	for {
		uuid, err := uuid()
		if err != nil {
			return "", err
		}
		id := sha.Sha1(uuid)[:8]

		if _, ok := a.metadatas[id]; !ok {
			return id, nil
		}
	}
}

func (a *app) CreateShare(filepath string, edit bool, password string, isDir bool, duration time.Duration) (string, error) {
	if !a.Enabled() {
		return "", fmt.Errorf("metadatas are disabled")
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	id, err := a.generateShareID()
	if err != nil {
		return "", err
	}

	a.metadatas[id] = provider.Share{
		ID:       id,
		Path:     filepath,
		RootName: path.Base(filepath),
		Edit:     edit,
		Password: password,
		File:     !isDir,
		Creation: a.clock.Now(),
		Duration: duration,
	}

	return id, a.saveMetadatas()
}

func (a *app) DeleteShare(id string) error {
	if !a.Enabled() {
		return fmt.Errorf("metadatas are disabled")
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	delete(a.metadatas, id)

	return a.saveMetadatas()
}

func (a *app) RenameSharePath(oldPath, newPath string) error {
	if !a.Enabled() {
		return nil
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	for id, share := range a.metadatas {
		if strings.HasPrefix(share.Path, oldPath) {
			share.Path = strings.Replace(share.Path, oldPath, newPath, 1)
			a.metadatas[id] = share
		}
	}

	return a.saveMetadatas()
}

func (a *app) DeleteSharePath(path string) error {
	if !a.Enabled() {
		return nil
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	for id, share := range a.metadatas {
		if strings.HasPrefix(share.Path, path) {
			delete(a.metadatas, id)
		}
	}

	return a.saveMetadatas()
}
