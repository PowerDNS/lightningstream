package memory

import (
	"testing"

	"powerdns.com/platform/lightningstream/storage/tester"
)

func TestBackend(t *testing.T) {
	b := New()
	tester.DoBackendTests(t, b)
}
