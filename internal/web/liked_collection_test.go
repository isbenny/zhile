package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLikedCollectionIsStableAndProtected(t *testing.T) {
	initCollectionDBForTest(t)
	router := newCollectionTestRouter()
	// An existing user playlist with the same display name must remain untouched.
	manual := Collection{Name: "我喜欢的音乐"}
	if err := db.Create(&manual).Error; err != nil {
		t.Fatal(err)
	}
	var likedID uint
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, RoutePrefix+"/collections/liked", nil)
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		router.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("create liked: %d %s", rec.Code, rec.Body.String())
		}
		var result struct {
			ID uint `json:"id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.ID == 0 || result.ID == manual.ID {
			t.Fatal("liked replaced an existing user playlist")
		}
		if i == 1 && likedID != result.ID {
			t.Fatal("duplicate liked collection")
		}
		likedID = result.ID
	}
	song := SavedSong{CollectionID: likedID, SongID: "test", Source: "local", Name: "Saved track"}
	if err := db.Create(&song).Error; err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, fmt.Sprintf("%s/collections/%d", RoutePrefix, likedID), bytes.NewBufferString(`{"name":"changed"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		router.ServeHTTP(rec, req)
		if rec.Code != 400 {
			t.Fatalf("%s liked: %d %s", method, rec.Code, rec.Body.String())
		}
	}
	if countSavedSongs(likedID) != 1 {
		t.Fatal("liked songs lost after rejected deletion")
	}
	var count int64
	db.Model(&Collection{}).Where("system_key = ?", "liked").Count(&count)
	if count != 1 {
		t.Fatalf("liked count %d", count)
	}
}

func TestManualCollectionCannotClaimLikedKey(t *testing.T) {
	initCollectionDBForTest(t)
	router := newCollectionTestRouter()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, RoutePrefix+"/collections", bytes.NewBufferString(`{"name":"Custom","system_key":"liked"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var collection Collection
	if err := db.First(&collection).Error; err != nil {
		t.Fatal(err)
	}
	if collection.SystemKey != nil {
		t.Fatal("client supplied system key persisted")
	}
}
