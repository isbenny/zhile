package web

import (
	"context"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guohuiyuan/music-lib/model"
	"github.com/guohuiyuan/music-lib/netease"
)

// Only public recommendations are cached. Personal history stays in the browser.
type homeFeedStore struct {
	mu      sync.Mutex
	items   []model.Playlist
	expires time.Time
	pending chan struct{}
	load    func() ([]model.Playlist, error)
}

func (s *homeFeedStore) get(ctx context.Context) []model.Playlist {
	s.mu.Lock()
	if time.Now().Before(s.expires) {
		items := append([]model.Playlist(nil), s.items...)
		s.mu.Unlock()
		return items
	}
	if s.pending == nil {
		s.pending = make(chan struct{})
		go func() {
			items, err := s.load()
			s.mu.Lock()
			defer s.mu.Unlock()
			if err == nil && len(items) > 0 {
				if len(items) > 8 {
					items = items[:8]
				}
				s.items = append([]model.Playlist(nil), items...)
				s.expires = time.Now().Add(10 * time.Minute)
			} else {
				s.expires = time.Now().Add(30 * time.Second)
			}
			close(s.pending)
			s.pending = nil
		}()
	}
	wait := s.pending
	s.mu.Unlock()
	select {
	case <-wait:
	case <-ctx.Done():
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]model.Playlist(nil), s.items...)
}

var publicHomeFeed = homeFeedStore{load: netease.GetRecommendedPlaylists}

func serveHomeFeed(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 6*time.Second)
	defer cancel()
	items := publicHomeFeed.get(ctx)
	type card struct {
		Name   string `json:"name"`
		Cover  string `json:"cover"`
		Source string `json:"source"`
		URL    string `json:"url"`
	}
	cards := make([]card, 0, len(items))
	for _, p := range items {
		if p.ID == "" || p.Name == "" {
			continue
		}
		p.Source = "netease"
		cards = append(cards, card{p.Name, p.Cover, p.Source, playlistDetailURL(RoutePrefix, "playlist", p)})
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(200, gin.H{"playlists": cards})
}
