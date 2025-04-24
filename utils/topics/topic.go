package topics

import (
	"sync"
)

// New returns a new Topic
func New[T any]() *Topic[T] {
	return &Topic[T]{
		subscribers: make(map[subscriptionID]chan<- T),
	}
}

// NewWithInitial returns a new Topic that is pre-seeded with a last value.
func NewWithInitial[T any](v T) *Topic[T] {
	return &Topic[T]{
		subscribers: make(map[subscriptionID]chan<- T),
		last:        v,
		hasLast:     true,
	}
}

// Topic is a single topic that subscribers can Subscribe() to
type Topic[T any] struct {
	mu          sync.Mutex
	subscribers map[subscriptionID]chan<- T
	lastID      subscriptionID
	last        T
	hasLast     bool
}

// Publish publishes a new value to all subscribers
func (t *Topic[T]) Publish(v T) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.last = v
	t.hasLast = true
	for _, ch := range t.subscribers {
		ch <- v // blocking
	}
}

// Last returns the last published value, if available
func (t *Topic[T]) Last() (value T, ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.hasLast {
		var zero T
		return zero, false
	}
	return t.last, true
}

// Subscribe creates a new Subscription.
// By default, this is an unbuffered channel.
// Is sendLast is set:
// - We will immediately send the last value, if any.
// - The channel will be a buffered one with size 1.
func (t *Topic[T]) Subscribe(sendLast bool) *Subscription[T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	var ch chan T
	if sendLast {
		ch = make(chan T, 1)
	} else {
		ch = make(chan T)
	}

	t.lastID++
	id := t.lastID

	t.subscribers[id] = ch

	if sendLast && t.hasLast {
		// Will not block, because the channel is buffered and nothing
		// else can publish into this while we hold the lock.
		ch <- t.last
	}

	sub := &Subscription[T]{
		id:    id,
		topic: t,
		ch:    ch,
	}
	return sub
}

// unsubscribeID is called by Subscription.Close()
// It removes a subscription.
func (t *Topic[T]) unsubscribeID(id subscriptionID) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ch, exists := t.subscribers[id]
	if !exists {
		return
	}
	close(ch)
	delete(t.subscribers, id)
}
