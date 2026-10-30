package web

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guohuiyuan/music-lib/model"
)

func TestHomeFeedCoalescesRequestsAndCachesResults(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	store := homeFeedStore{load: func() ([]model.Playlist, error) {
		calls.Add(1)
		<-release
		return []model.Playlist{{ID: "one", Name: "A"}}, nil
	}}
	done := make(chan []model.Playlist, 2)
	go func() { done <- store.get(context.Background()) }()
	go func() { done <- store.get(context.Background()) }()
	close(release)
	for i := 0; i < 2; i++ {
		if items := <-done; len(items) != 1 || items[0].ID != "one" {
			t.Fatalf("unexpected feed: %#v", items)
		}
	}
	store.get(context.Background())
	if calls.Load() != 1 {
		t.Fatalf("expected one upstream request, got %d", calls.Load())
	}
}

func TestHomeFeedCanceledVisitorDoesNotBlockOtherVisitors(t *testing.T) {
	release := make(chan struct{})
	store := homeFeedStore{load: func() ([]model.Playlist, error) { <-release; return []model.Playlist{{ID: "one", Name: "A"}}, nil }}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if items := store.get(ctx); len(items) != 0 {
		t.Fatal("first load should not fabricate recommendations")
	}
	if time.Since(start) > time.Second {
		t.Fatal("canceled request blocked on upstream")
	}
	close(release)
	if items := store.get(context.Background()); len(items) != 1 {
		t.Fatal("background result should remain available")
	}
}

func TestHomeFeedKeepsLastGoodResultOnFailure(t *testing.T) {
	store := homeFeedStore{items: []model.Playlist{{ID: "saved", Name: "A"}}, load: func() ([]model.Playlist, error) { return nil, errors.New("offline") }}
	items := store.get(context.Background())
	if len(items) != 1 || items[0].ID != "saved" {
		t.Fatalf("lost last good recommendation: %#v", items)
	}
}
