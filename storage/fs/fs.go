package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/storage"
)

type Backend struct {
	rootPath string
}

func (b *Backend) List(ctx context.Context, prefix string) (storage.BlobList, error) {
	var blobs storage.BlobList

	entries, err := os.ReadDir(b.rootPath)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if strings.HasSuffix(name, ".tmp") {
			continue
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			if os.IsNotExist(err) {
				continue // could have been removed in the meantime
			}
			return nil, err
		}
		blobs = append(blobs, storage.Blob{
			Name: name,
			Size: info.Size(),
		})
	}

	sort.Slice(blobs, func(i, j int) bool {
		return blobs[i].Name < blobs[j].Name
	})
	return blobs, nil
}

func (b *Backend) Load(ctx context.Context, name string) ([]byte, error) {
	if strings.Contains(name, "/") {
		return nil, os.ErrNotExist
	}
	fullPath := filepath.Join(b.rootPath, name)
	return os.ReadFile(fullPath)
}

func (b *Backend) Store(ctx context.Context, name string, data []byte) error {
	if strings.Contains(name, "/") {
		return os.ErrPermission
	}
	fullPath := filepath.Join(b.rootPath, name)
	tmpPath := fullPath + ".tmp" // ignored by List()
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, fullPath)
}

func New(rootPath string) (*Backend, error) {
	if rootPath == "" {
		return nil, fmt.Errorf("storage.root_path must be set for the fs backend")
	}
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		return nil, err
	}
	b := &Backend{rootPath: rootPath}
	return b, nil
}

func init() {
	storage.RegisterBackend("fs", func(st config.Storage) (storage.Interface, error) {
		return New(st.RootPath)
	})
}
