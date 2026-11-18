package api

import (
	"context"
	"testing"
	"time"
)

// startHub runs a hub for the length of the test.
func startHub(t *testing.T, bufferSize int) *Hub {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	hub := NewHub(bufferSize)
	done := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return hub
}

// receive waits briefly for one event.
func receive(t *testing.T, events <-chan int64) (version int64, open bool) {
	t.Helper()
	select {
	case v, ok := <-events:
		return v, ok
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for an event")
		return 0, false
	}
}

func TestHubDeliversAPublishedVersionToEverySubscriber(t *testing.T) {
	hub := startHub(t, 4)
	a, cancelA, err := hub.Subscribe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cancelA()
	b, cancelB, err := hub.Subscribe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cancelB()

	hub.Publish(7)

	for name, events := range map[string]<-chan int64{"a": a, "b": b} {
		if v, ok := receive(t, events); !ok || v != 7 {
			t.Errorf("subscriber %s got %d (open %v), want 7", name, v, ok)
		}
	}
}

func TestHubStopsDeliveringAfterACancel(t *testing.T) {
	hub := startHub(t, 4)
	events, cancel, err := hub.Subscribe(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	cancel()
	cancel() // cancelling twice must be harmless
	hub.Publish(1)

	if _, ok := receive(t, events); ok {
		t.Error("the channel of a cancelled subscription should be closed")
	}
	if got := hub.subscriberCount(); got != 0 {
		t.Errorf("%d subscribers left, want 0", got)
	}
}

func TestHubDropsASlowSubscriberWithoutBlockingTheOthers(t *testing.T) {
	hub := startHub(t, 2)
	slow, cancelSlow, _ := hub.Subscribe(context.Background())
	defer cancelSlow()
	fast, cancelFast, _ := hub.Subscribe(context.Background())
	defer cancelFast()

	// The slow subscriber never reads; the fast one keeps up. Its buffer holds 2 events, so
	// the third publish finds it full.
	for version := int64(1); version <= 5; version++ {
		hub.Publish(version)
		if v, ok := receive(t, fast); !ok || v != version {
			t.Fatalf("fast subscriber got %d (open %v), want %d", v, ok, version)
		}
	}

	// The slow subscriber was cut off: it can still read what was buffered, then sees "closed".
	received := 0
	for {
		_, ok := receive(t, slow)
		if !ok {
			break
		}
		received++
	}
	if received != 2 {
		t.Errorf("the slow subscriber read %d buffered events, want 2", received)
	}
	if got := hub.subscriberCount(); got != 1 {
		t.Errorf("%d subscribers left, want only the fast one", got)
	}
}

func TestHubClosesSubscribersWhenItStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	hub := NewHub(4)
	done := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(done)
	}()
	events, _, err := hub.Subscribe(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	cancel()
	<-done

	if _, ok := receive(t, events); ok {
		t.Error("the subscriber's channel should be closed once the hub stops")
	}
	// None of these may block forever now that nobody is listening.
	hub.Publish(1)
	if _, _, err := hub.Subscribe(context.Background()); err == nil {
		t.Error("Subscribe on a stopped hub succeeded, want an error")
	}
}

func TestSubscribeGivesUpWhenItsContextIsDone(t *testing.T) {
	hub := NewHub(4) // never started, so nothing will ever accept the subscription
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, _, err := hub.Subscribe(ctx)

	if err == nil {
		t.Error("Subscribe succeeded on a hub that is not running")
	}
}
