package lmdbenv

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/c2h5oh/datasize"
)

const (
	DefaultDirMask  = 0775
	DefaultFileMask = 0664
	DefaultMapSize  = 1 * datasize.GB
	DefaultMaxDBs   = 64
)

// Options are used for NewWithOptions, allowing a user to override them
// This type is also used for the yaml config file.
type Options struct {
	DirMask  os.FileMode       `yaml:"dir_mask"`
	FileMask os.FileMode       `yaml:"file_mask"`
	MapSize  datasize.ByteSize `yaml:"map_size"`
	MaxDBs   int               `yaml:"max_dbs"`
	NoSubdir bool              `yaml:"no_subdir"`
	Create   bool              `yaml:"create"`
	EnvFlags uint              `yaml:"-"` // Too dangerous for direct yaml support
}

// WithDefaults returns new Options with defaults set for values that were not set
func (o Options) WithDefaults() Options {
	// Note that we receive a copy of o
	if o.DirMask == 0 {
		o.DirMask = DefaultDirMask
	}
	if o.FileMask == 0 {
		o.FileMask = DefaultFileMask
	}
	if o.MaxDBs == 0 {
		o.MaxDBs = DefaultMaxDBs
	}
	return o
}

// New creates an LMDB Env suitable for generating a filter platform
// production LMDB test database. The returned env must be closed after use.
func New(path string, flags uint) (*lmdb.Env, error) {
	return NewWithOptions(path, Options{EnvFlags: flags})
}

// NewWithOptions creates an LMDB Env suitable for generating a filter platform
// production LMDB test database using given options.
// The returned env must be closed after use.
func NewWithOptions(path string, opt Options) (*lmdb.Env, error) {
	opt = opt.WithDefaults()
	env, err := lmdb.NewEnv()
	if err != nil {
		return nil, fmt.Errorf("lmdb env: new: %v", err)
	}

	if opt.Create {
		opt.EnvFlags |= lmdb.Create
	}

	// Special default MapSize handling: only if we passed the create flag
	// will we set a default value for the MapSize, otherwise we leave it
	// at 0, so that LMDB automatically detects this value from the existing
	// file. Otherwise the query tool can bump the LMDB MapSize. Closes #1301.
	createIsSet := false
	if opt.EnvFlags&lmdb.Create > 0 {
		createIsSet = true
	}

	if opt.NoSubdir {
		opt.EnvFlags |= lmdb.NoSubdir
	}

	if createIsSet {
		dirPath := path
		if opt.EnvFlags&lmdb.NoSubdir > 0 {
			dirPath, _ = filepath.Split(dirPath)
		}
		err := os.MkdirAll(dirPath, opt.DirMask)
		if err != nil {
			return nil, fmt.Errorf("lmdb env: mkdir: %v", err)
		}
	}

	mapSize := opt.MapSize
	if mapSize == 0 && createIsSet {
		mapSize = DefaultMapSize
	}
	err = env.SetMapSize(int64(mapSize))
	if err != nil {
		return nil, fmt.Errorf("lmdb env: setmapsize: %v", err)
	}

	err = env.SetMaxDBs(opt.MaxDBs)
	if err != nil {
		return nil, fmt.Errorf("lmdb env: setmaxdbs: %v", err)
	}
	err = env.Open(path, opt.EnvFlags, opt.FileMask)
	if err != nil {
		return nil, fmt.Errorf("lmdb env: open: %v", err)
	}
	return env, nil
}
