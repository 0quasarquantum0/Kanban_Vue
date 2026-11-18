package api

import (
	"context"
	"errors"
	"sync"
)

// Hub tells every connected client when the state's version changes.
//
// One goroutine (Run) owns the set of subscribers. Nothing else touches that set: other
// goroutines ask it to subscribe, unsubscribe or publish by sending on channels, and Run's
// select handles one request at a time. This is Go's "share memory by communicating": no mutex
// guards the set, because only one goroutine ever reads or writes it.
type Hub struct {
	bufferSize int

	subscribe   chan *subscription
	unsubscribe chan *subscription
	publish     chan int64
	count       chan chan int // test hook: asks Run how many subscribers it has

	stopped chan struct{} // closed when Run returns
}

// subscription is one client's mailbox.
type subscription struct {
	events chan int64
	once   sync.Once
}

// NewHub returns a hub whose subscribers can each hold bufferSize unread events. Call Run to
// start it.
func NewHub(bufferSize int) *Hub {
	return &Hub{
		bufferSize:  bufferSize,
		subscribe:   make(chan *subscription),
		unsubscribe: make(chan *subscription),
		publish:     make(chan int64),
		count:       make(chan chan int),
		stopped:     make(chan struct{}),
	}
}

// errHubStopped is returned by Subscribe when the hub is no longer running.
var errHubStopped = errors.New("hub stopped")

// Run serves the hub until ctx is cancelled, then closes every subscriber's channel.
func (h *Hub) Run(ctx context.Context) {
	defer close(h.stopped)
	subscribers := map[*subscription]struct{}{}

	drop := func(s *subscription) {
		if _, ok := subscribers[s]; ok {
			delete(subscribers, s)
			close(s.events)
		}
	}

	for {
		select {
		case <-ctx.Done():
			for s := range subscribers {
				drop(s)
			}
			return

		case s := <-h.subscribe:
			subscribers[s] = struct{}{}

		case s := <-h.unsubscribe:
			drop(s)

		case version := <-h.publish:
			for s := range subscribers {
				select {
				case s.events <- version:
				default:
					// This client's buffer is full: it is not reading. Cut it off instead of
					// making every other client wait. It reconnects and catches up.
					drop(s)
				}
			}

		case reply := <-h.count:
			reply <- len(subscribers)
		}
	}
}

// Subscribe registers a new client. The returned channel receives each published version; it is
// closed when the client is cancelled, dropped for being slow, or the hub stops. Call cancel
// when done (it is safe to call more than once).
func (h *Hub) Subscribe(ctx context.Context) (events <-chan int64, cancel func(), err error) {
	s := &subscription{events: make(chan int64, h.bufferSize)}
	select {
	case h.subscribe <- s:
	case <-h.stopped:
		return nil, nil, errHubStopped
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}

	cancel = func() {
		s.once.Do(func() {
			select {
			case h.unsubscribe <- s:
			case <-h.stopped: // nothing left to tell
			}
		})
	}
	return s.events, cancel, nil
}

// Publish announces that the state is now at version. It never blocks for long, and does
// nothing once the hub has stopped.
func (h *Hub) Publish(version int64) {
	select {
	case h.publish <- version:
	case <-h.stopped:
	}
}

// subscriberCount reports how many clients are subscribed. It exists for tests.
func (h *Hub) subscriberCount() int {
	reply := make(chan int)
	select {
	case h.count <- reply:
		return <-reply
	case <-h.stopped:
		return 0
	}
}
