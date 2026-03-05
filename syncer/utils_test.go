package syncer

import (
	"regexp"
	"testing"

	"github.com/PowerDNS/lightningstream/config"
	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/simpleblob/backends/memory"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var reGenerationID = regexp.MustCompile(`^G-[0-9a-f]{16}$`)

// TestGenerationID_Format verifies that generationID returns a string matching
// the expected format: "G-" followed by exactly 16 lowercase hex digits.
// The format is part of the snapshot filename spec and must not regress to the
// old hardcoded "GX" value.
func TestGenerationID_Format(t *testing.T) {
	l, _ := test.NewNullLogger()
	err := lmdbenv.TestEnv(func(env *lmdb.Env) error {
		s, err := New("test", env, memory.New(), config.Config{}, config.LMDB{}, Options{})
		require.NoError(t, err)
		s.l = l

		gid := s.generationID()
		assert.Regexp(t, reGenerationID, gid,
			"generationID must match G-<16 hex digits>, got %q", gid)
		return nil
	})
	require.NoError(t, err)
}

// TestGenerationID_UniquePerInstance verifies that two independently created
// Syncer instances receive different generation IDs. The ID is seeded from
// time.Now().UnixNano() at construction, so two instances created in the same
// process should never collide.
func TestGenerationID_UniquePerInstance(t *testing.T) {
	l, _ := test.NewNullLogger()
	var gid1, gid2 string

	err := lmdbenv.TestEnv(func(env *lmdb.Env) error {
		s1, err := New("test", env, memory.New(), config.Config{}, config.LMDB{}, Options{})
		require.NoError(t, err)
		s1.l = l
		gid1 = s1.generationID()

		s2, err := New("test", env, memory.New(), config.Config{}, config.LMDB{}, Options{})
		require.NoError(t, err)
		s2.l = l
		gid2 = s2.generationID()
		return nil
	})
	require.NoError(t, err)

	assert.NotEqual(t, gid1, gid2,
		"two Syncer instances must have different generation IDs")
}
