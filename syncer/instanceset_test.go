package syncer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceSet_CleanDisappeared(t *testing.T) {
	s := NewInstanceSet()
	s.Add("z")
	s.Add("x")
	s.Add("foo")
	s.Add("y")
	s.Add("bar")
	cleaned := s.CleanDisappeared([]string{"q", "x", "y", "z"})
	assert.Equal(t, []string{"bar", "foo"}, cleaned)
	assert.Equal(t, []string{"x", "y", "z"}, s.List())
}

func TestInstanceSet_Contains(t *testing.T) {
	s := NewInstanceSet()
	assert.False(t, s.Contains("foo"))
	s.Add("foo")
	assert.True(t, s.Contains("foo"))
}

func TestInstanceSet_Done(t *testing.T) {
	s := NewInstanceSet()
	assert.True(t, s.Done())
	s.Add("foo")
	assert.False(t, s.Done())
	s.Add("bar")
	assert.False(t, s.Done())
	s.Remove("foo")
	assert.False(t, s.Done())
	s.Remove("bar")
	assert.True(t, s.Done())
}

func TestInstanceSet_List(t *testing.T) {
	s := NewInstanceSet()
	s.Add("z")
	s.Add("x")
	s.Add("foo")
	s.Add("y")
	s.Add("bar")
	// Deterministic sorted output
	for i := 0; i < 10; i++ {
		assert.Equal(t, []string{"bar", "foo", "x", "y", "z"}, s.List())
	}
}

func TestInstanceSet_Remove(t *testing.T) {
	s := NewInstanceSet()
	assert.False(t, s.Contains("foo"))
	assert.False(t, s.Contains("bar"))
	assert.False(t, s.Contains("xyz"))
	s.Add("foo")
	s.Add("bar")
	assert.True(t, s.Contains("foo"))
	assert.True(t, s.Contains("bar"))
	assert.False(t, s.Contains("xyz"))
	s.Remove("foo")
	assert.False(t, s.Contains("foo"))
	assert.True(t, s.Contains("bar"))
}

func TestInstanceSet_String(t *testing.T) {
	s := NewInstanceSet()
	s.Add("z")
	s.Add("x")
	s.Add("foo")
	s.Add("y")
	s.Add("bar")
	// Deterministic sorted output
	for i := 0; i < 10; i++ {
		assert.Equal(t, "bar foo x y z", s.String())
	}
}
