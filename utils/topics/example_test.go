package topics_test

import (
	"fmt"
	"sync"

	"github.com/PowerDNS/lightningstream/utils/topics"
)

func Example() {
	t := topics.New[int]()
	// No listeners yet to receive this
	t.Publish(1)

	// But you can access the last value
	if last, ok := t.Last(); ok {
		fmt.Printf("last=%v\n", last)
	}

	// Needed for test serialization
	var wg sync.WaitGroup
	wg.Add(1)

	// Add a subscriber
	sub := t.Subscribe(false)
	go func() {
		defer wg.Done()
		ch := sub.Channel()
		for {
			m, ok := <-ch
			if !ok {
				fmt.Println("channel closed")
				return // channel closed
			}
			fmt.Printf("received=%d\n", m)
		}
	}()

	t.Publish(2)
	t.Publish(3)

	// Close subscription
	sub.Close()
	wg.Wait()

	// Never received
	t.Publish(4)

	// But you can access the last value
	if last, ok := t.Last(); ok {
		fmt.Printf("last=%v\n", last)
	}

	// Output:
	// last=1
	// received=2
	// received=3
	// channel closed
	// last=4
}
