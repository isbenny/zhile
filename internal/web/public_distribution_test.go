package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPreviewDiscoveryPreservesDownloadRestrictions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, path := range []string{"/search", "/home_feed", "/recommend", "/playlist_categories", "/category_playlists", "/toplists", "/toplist_songs", "/playlist", "/album", "/album_jump", "/download", "/switch_source", "/api/downloads/queue/container", "/cookies"} {
		t.Run(path, func(t *testing.T) {
			router := gin.New()
			api := router.Group(RoutePrefix)
			api.Use(distributionGuard(true))
			called := false
			api.GET(path, func(c *gin.Context) {
				called = true
				if c.GetBool("PublicLocalOnly") {
					t.Error("preview still hides online discovery")
				}
				c.Status(204)
			})
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest("GET", RoutePrefix+path+"?source=qq", nil))
			blocked := path == "/download" || path == "/switch_source" || path == "/api/downloads/queue/container" || path == "/cookies"
			if called == blocked {
				t.Fatalf("called=%t status=%d body=%s", called, rec.Code, rec.Body.String())
			}
			csp := rec.Header().Get("Content-Security-Policy")
			if !strings.Contains(csp, "img-src 'self' data: blob: https: http:") || !strings.Contains(csp, "media-src 'self' blob:;") {
				t.Fatalf("unexpected preview CSP: %s", csp)
			}
		})
	}
}

func TestPublicEditionBlocksOnlineRoutesAndSourceSpoofing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		method, path, query string
		allowed             bool
	}{
		{"GET", "/recommend", "", false}, {"GET", "/playlist", "?source=netease&id=1", false},
		{"GET", "/switch_source", "?source=local&target=qq", false}, {"GET", "/cover_proxy", "?url=https://example.com/art", false},
		{"GET", "/download_cover", "?source=local&url=https://example.com/art", false},
		{"GET", "/download", "?source=qq&id=1", false}, {"POST", "/download", "?source=netease&id=1", false},
		{"GET", "/download", "?source=local&id=local-file", true}, {"GET", "/lyric", "?source=local-file&id=1", true},
		{"GET", "/inspect", "?source=qq&id=1", false}, {"POST", "/local_music/auto_cache", "", false},
		{"POST", "/api/downloads/queue", "", false}, {"POST", "/api/downloads/queue/container", "", false},
		{"POST", "/api/downloads/queue/:id/:action", "", false}, {"POST", "/qr_login/:source", "", false},
		{"POST", "/videogen/init", "", false}, {"POST", "/collections/import", "", false},
		{"POST", "/local_music/upload", "", true}, {"GET", "/local_music", "", true},
	} {
		t.Run(test.method+test.path+test.query, func(t *testing.T) {
			router := gin.New()
			group := router.Group(RoutePrefix)
			group.Use(publicDistributionGuard())
			called := false
			group.Handle(test.method, test.path, func(c *gin.Context) { called = true; c.Status(204) })
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(test.method, RoutePrefix+test.path+test.query, nil))
			if called != test.allowed {
				t.Fatalf("called=%t status=%d body=%s", called, rec.Code, rec.Body.String())
			}
			if !test.allowed && rec.Code != 403 {
				t.Fatalf("denied status %d", rec.Code)
			}
		})
	}
}

func TestPublicEditionPreservesLocalCollectionWrites(t *testing.T) {
	initCollectionDBForTest(t)
	manual := Collection{Name: "Local library"}
	db.Create(&manual)
	imported := Collection{Name: "Old online list", Kind: collectionKindImported, Source: "qq", ExternalID: "not-to-fetch"}
	db.Create(&imported)
	router := gin.New()
	api := router.Group(RoutePrefix)
	api.Use(publicDistributionGuard())
	RegisterCollectionRoutes(api)
	for _, source := range []string{"qq", "local"} {
		rec := httptest.NewRecorder()
		body := fmt.Sprintf(`{"id":"test-file","source":%q,"name":"Local song"}`, source)
		req := httptest.NewRequest("POST", fmt.Sprintf("%s/collections/%d/songs", RoutePrefix, manual.ID), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		want := 403
		if source == "local" {
			want = 200
		}
		if rec.Code != want {
			t.Fatalf("%s: %d %s", source, rec.Code, rec.Body.String())
		}
	}
	if countSavedSongs(manual.ID) != 1 {
		t.Fatal("local song was not saved")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("%s/collections/%d/songs", RoutePrefix, imported.ID), nil))
	if rec.Code != 403 {
		t.Fatal("imported playlist could fetch remote songs")
	}
	var found Collection
	if db.First(&found, imported.ID).Error != nil {
		t.Fatal("existing imported collection was deleted")
	}
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", RoutePrefix+"/collections", nil))
	var records []Collection
	if json.Unmarshal(rec.Body.Bytes(), &records) != nil {
		t.Fatal("collections response invalid")
	}
}
