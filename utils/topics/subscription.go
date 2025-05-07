package topics

import (
	"context"
	"io"
	"sync"
)

// subscriptionID in an internal ID for every new subscription.
// This is an internal that is only valid within a single Topic
type subscriptionID uint

// Subscription is the reference to a Topic subscription.
// When the publisher writes to a Topic, it will BLOCK until all subscribers
// have received the message.
// Subscription MUST always be closed with Close() when no longer used.
type Subscription[T any] struct {
	id subscriptionID

	mu    sync.Mutex
	topic *Topic[T]
	ch    <-chan T
}

// Channel returns the chan that can be used to receive values from this
// subscription.
func (s *Subscription[T]) Channel() <-chan T {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ch
}

// Next is a convenience method to read the next value. It blocks until
// the next value is available, or until the context is closed.
// It returns an error if the context or channel was closed.
func (s *Subscription[T]) Next(ctx context.Context) (value T, err error) {
	var zero T
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case v, ok := <-s.Channel():
		if !ok {
			return zero, io.ErrClosedPipe
		}
		return v, nil
	}
}

// Close terminates this subscription. The Topic will close the channel.
// Close can safely be called multiple times, even from different goroutines.
func (s *Subscription[T]) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.topic == nil {
		return // already closed
	}

	s.topic.unsubscribeID(s.id)
	s.ch = nil
	s.topic = nil
}
