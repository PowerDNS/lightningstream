package memory

import (
	"context"
	"os"
	"sort"
	"strings"
	"sync"

	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/storage"
)

type Backend struct {
	mu    sync.Mutex
	blobs map[string][]byte
}

func (b *Backend) List(ctx context.Context, prefix string) (storage.BlobList, error) {
	var blobs storage.BlobList

	b.mu.Lock()
	for name, data := range b.blobs {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		blobs = append(blobs, storage.Blob{
			Name: name,
			Size: int64(len(data)),
		})
	}
	b.mu.Unlock()

	sort.Sort(blobs)
	return blobs, nil
}

func (b *Backend) Load(ctx context.Context, name string) ([]byte, error) {
	b.mu.Lock()
	data, exists := b.blobs[name]
	b.mu.Unlock()

	if !exists {
		return nil, os.ErrNotExist
	}
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data) // safe, because data was a copy itself
	return dataCopy, nil
}

func (b *Backend) Store(ctx context.Context, name string, data []byte) error {
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	b.mu.Lock()
	b.blobs[name] = dataCopy
	b.mu.Unlock()

	return nil
}

func New() *Backend {
	return &Backend{blobs: make(map[string][]byte)}
}

func init() {
	storage.RegisterBackend("memory", func(st config.Storage) (storage.Interface, error) {
		return New(), nil
	})
}
