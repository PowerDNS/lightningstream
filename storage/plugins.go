package storage

import (
	"context"
	"fmt"
	"strings"

	"powerdns.com/platform/lightningstream/config"
)

type Blob struct {
	Name string
	Size int64
}

type BlobList []Blob

func (bl BlobList) Len() int {
	return len(bl)
}

func (bl BlobList) Less(i, j int) bool {
	return bl[i].Name < bl[j].Name
}

func (bl BlobList) Swap(i, j int) {
	bl[i], bl[j] = bl[j], bl[i]
}

func (bl BlobList) Names() []string {
	var names []string
	for _, b := range bl {
		names = append(names, b.Name)
	}
	return names
}

func (bl BlobList) WithPrefix(prefix string) (blobs BlobList) {
	for _, b := range bl {
		if !strings.HasPrefix(b.Name, prefix) {
			continue
		}
		blobs = append(blobs, b)
	}
	return blobs
}

// Interface defines the interface storage plugins need to implement
type Interface interface {
	List(ctx context.Context, prefix string) (BlobList, error)
	Load(ctx context.Context, name string) ([]byte, error)
	Store(ctx context.Context, name string, data []byte) error
}

type InitFunc func(st config.Storage) (Interface, error)

var backends = make(map[string]InitFunc)

func RegisterBackend(typeName string, initFunc InitFunc) {
	backends[typeName] = initFunc
}

func GetBackend(sc config.Storage) (Interface, error) {
	if sc.Type == "" {
		return nil, fmt.Errorf("no storage.type configured")
	}
	initFunc, exists := backends[sc.Type]
	if !exists {
		return nil, fmt.Errorf("storage.type %q not found or registered", sc.Type)
	}
	return initFunc(sc)
}
