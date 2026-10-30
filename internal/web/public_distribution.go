package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/guohuiyuan/music-lib/model"
)

// The owner selected the normal app for v0.2.1. The optional restricted mode
// remains available in source but is not enabled in this release.
const publicLocalOnly = false
const publicRightsMessage = "公开版仅支持本地自有或已获授权音乐，尚未接入获得授权的在线音源。"

func publicDistributionGuard() gin.HandlerFunc {
	return distributionGuard(false)
}

func distributionGuard(onlineDiscovery bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("PublicLocalOnly", !onlineDiscovery)
		// Prevent legacy remote artwork and media URLs in saved metadata from making
		// requests directly from the WebView. Music files are served by the local app.
		images := "'self' data: blob:"
		if onlineDiscovery {
			images += " https: http:"
		}
		c.Header("Content-Security-Policy", "img-src "+images+"; media-src 'self' blob:; connect-src 'self'; frame-src 'none'; object-src 'none'; base-uri 'self'")
		path := strings.TrimPrefix(c.FullPath(), RoutePrefix)
		if onlineDiscovery && c.Request.Method == http.MethodGet {
			switch path {
			case "/search", "/home_feed", "/recommend", "/playlist_categories", "/category_playlists", "/toplists", "/toplist_songs", "/playlist", "/album", "/album_jump":
				c.Next()
				return
			}
		}
		deny := func() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": publicRightsMessage, "code": "content_not_authorized"})
		}
		switch path {
		case "/", "/setup", "/login", "/logout", "/healthz", "/settings", "/my_collections", "/local_music_page", "/local_music", "/local_music/upload", "/local_music/duplicates", "/local_music/batch_match", "/local_music/reindex", "/local_music/:id", "/local_music/cover", "/collections", "/collections/liked", "/collections/:id", "/collections/:id/local_music", "/collections/:id/local_music/batch", "/api/downloads/records":
			c.Next()
			return
		case "/cookies":
			// Compatibility with the existing settings form; no cookies are read/saved.
			if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
				c.AbortWithStatusJSON(200, gin.H{})
				return
			}
			c.AbortWithStatusJSON(200, gin.H{"status": "disabled_in_public_edition"})
			return
		case "/app_update/check":
			// Do not let query parameters turn the official update check into a proxy.
			c.Request.URL.RawQuery = "repo=https%3A%2F%2Fgithub.com%2Fisbenny%2Fzhile&use_proxy=0"
			c.Next()
			return
		case "/search":
			keyword := strings.TrimSpace(c.Query("q"))
			var songs []model.Song
			var playlists []model.Playlist
			kind := c.DefaultQuery("type", "song")
			if kind == "playlist" {
				for _, p := range localCollectionSearchPlaylists(keyword) {
					if playlistExtraValue(p, "collection_kind") != "imported" {
						playlists = append(playlists, p)
					}
				}
			} else {
				kind = "song"
				songs = localMusicSearchSongs(keyword, 200)
			}
			renderIndex(c, songs, playlists, keyword, []string{"local"}, "", kind, "", "", "", false, "", nil)
			c.Abort()
			return
		case "/home_feed":
			c.AbortWithStatusJSON(200, gin.H{"playlists": []interface{}{}, "mode": "local"})
			return
		case "/download", "/download_lrc", "/lyric", "/inspect":
			if !isLocalMusicSource(c.Query("source")) {
				deny()
				return
			}
			c.Next()
			return
		case "/collection", "/collections/:id/songs", "/collections/:id/songs/batch", "/collections/:id/songs/:songID":
			id := c.Param("id")
			if path == "/collection" {
				id = c.Query("id")
			}
			collection, err := loadCollection(id)
			if err != nil {
				c.Next()
				return
			} // Existing route provides its normal 404.
			if collection.isImported() {
				if path == "/collection" {
					renderIndex(c, nil, nil, "", []string{"local"}, publicRightsMessage, "song", "", "", "", false, "", nil)
					c.Abort()
					return
				}
				deny()
				return
			}
			if c.Request.Method == http.MethodPost && strings.Contains(path, "/songs") {
				c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<20)
				var payload map[string]json.RawMessage
				if err := json.NewDecoder(c.Request.Body).Decode(&payload); err != nil {
					c.AbortWithStatusJSON(400, gin.H{"error": "无效的歌曲信息"})
					return
				}
				var sources []struct {
					Source string `json:"source"`
				}
				if raw, ok := payload["songs"]; ok {
					if json.Unmarshal(raw, &sources) != nil {
						deny()
						return
					}
				} else {
					var source string
					_ = json.Unmarshal(payload["source"], &source)
					sources = append(sources, struct {
						Source string `json:"source"`
					}{source})
				}
				for _, song := range sources {
					if !isLocalMusicSource(song.Source) {
						deny()
						return
					}
				}
				// Restore JSON for handlers that use ShouldBindJSON rather than body caching.
				raw, _ := json.Marshal(payload)
				c.Request.Body = io.NopCloser(bytes.NewReader(raw))
			}
			c.Next()
			return
		case "/api/downloads/queue", "/api/downloads/queue/:id", "/api/downloads/queue/:id/:action":
			if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodDelete {
				c.Next()
				return
			}
			deny()
			return
		default:
			deny()
			return
		}
	}
}
