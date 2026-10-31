package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guohuiyuan/go-music-dl/core"
	"golang.org/x/crypto/bcrypt"
)

func memberTestSetup(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	initCollectionDBForTest(t)
	resetAuthRuntimeForTest()
	gin.SetMode(gin.TestMode)
	h, _ := bcrypt.GenerateFromPassword([]byte("admin-test-password"), bcrypt.MinCost)
	settings := core.WebAuthSettings{Username: "owner", PasswordHash: string(h), SessionSecret: "test-secret"}
	if err := core.SaveWebAuthSettings(settings); err != nil {
		t.Fatal(err)
	}
	token, _ := createSessionValue(settings, time.Now())
	r := gin.New()
	r.SetHTMLTemplate(newTestTemplate(t))
	r.Use(func(c *gin.Context) { c.Set("PrivateMode", true); c.Set("SecureCookies", true) })
	api := r.Group(RoutePrefix)
	config := bindAuthMiddleware(api, StartOptions{RequireAuthForAll: true})
	RegisterCollectionRoutes(api)
	for _, p := range []string{"/search", "/download", "/local_music", "/local_music/cover", "/user_playlists", "/settings", "/videos/file", "/api/downloads/queue"} {
		api.GET(p, func(c *gin.Context) { c.String(200, "allowed") })
	}
	config.GET("/cookies", func(c *gin.Context) { c.String(200, "secret") })
	return r, token
}
func memberReq(r *gin.Engine, method, path, token, body, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://music.example/music"+path, strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "https://music.example")
	req.Header.Set("Content-Type", contentType)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
func testInvite(t *testing.T, code string, expires time.Time) {
	t.Helper()
	if err := db.Create(&WebInvitation{TokenHash: tokenDigest(code), ExpiresAt: expires}).Error; err != nil {
		t.Fatal(err)
	}
}
func testMember(t *testing.T, name string) (WebMember, string) {
	t.Helper()
	testInvite(t, name+"-invite", time.Now().Add(time.Hour))
	m, e := registerMember(name+"-invite", name, "test-password-123")
	if e != nil {
		t.Fatal(e)
	}
	token, e := createSessionValue(m.auth(), time.Now())
	if e != nil {
		t.Fatal(e)
	}
	return m, token
}
func TestInvitationRegistrationAndReplay(t *testing.T) {
	r, _ := memberTestSetup(t)
	testInvite(t, "first-invite", time.Now().Add(time.Hour))
	body := url.Values{"username": {"friend_one"}, "password": {"my-test-pass-123"}, "password_confirm": {"my-test-pass-123"}, "invite_code": {"first-invite"}}
	w := memberReq(r, "POST", "/register", "", body.Encode(), "application/x-www-form-urlencoded")
	if w.Code != 303 {
		t.Fatalf("registration: %d %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Header().Get("Set-Cookie"), "Secure") || !strings.Contains(w.Header().Get("Set-Cookie"), "HttpOnly") {
		t.Fatal("missing secure cookie")
	}
	if _, e := registerMember("first-invite", "friend_two", "test-password-123"); e == nil {
		t.Fatal("reused invitation")
	}
	testInvite(t, "expired", time.Now().Add(-time.Hour))
	if _, e := registerMember("expired", "other_friend", "test-password-123"); e == nil {
		t.Fatal("expired invitation")
	}
	testInvite(t, "reserve", time.Now().Add(time.Hour))
	if _, e := registerMember("reserve", "OWNER", "test-password-123"); e == nil {
		t.Fatal("admin name collision")
	}
	if _, e := registerMember("reserve", "friend_one", "test-password-123"); e == nil {
		t.Fatal("duplicate name")
	}
	if _, e := registerMember("reserve", "friend_three", "test-password-123"); e != nil {
		t.Fatalf("failed registration consumed invitation: %v", e)
	}
	var inv WebInvitation
	db.Where("token_hash = ?", tokenDigest("first-invite")).First(&inv)
	if inv.UsedBy == 0 {
		t.Fatal("invite not redeemed")
	}
}
func TestConcurrentInvitationIsSingleUse(t *testing.T) {
	memberTestSetup(t)
	testInvite(t, "concurrent", time.Now().Add(time.Hour))
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, name := range []string{"first_person", "second_person"} {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			_, err := registerMember("concurrent", n, "test-password-123")
			results <- err
		}(name)
	}
	wg.Wait()
	close(results)
	success := 0
	for e := range results {
		if e == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("registrations=%d", success)
	}
}
func TestMemberCollectionIsolationAndAdminBoundary(t *testing.T) {
	r, admin := memberTestSetup(t)
	a, ta := testMember(t, "alice")
	b, tb := testMember(t, "bobby")
	legacy := Collection{Name: "owner legacy collection"}
	db.Create(&legacy)
	// Spoofed primary key, ownership, and nested associations must be discarded.
	payload := fmt.Sprintf(`{"id":%d,"name":"Alice private","owner_id":%d,"SavedSongs":[{"song_id":"spoof"}]}`, legacy.ID, b.ID)
	w := memberReq(r, "POST", "/collections", ta, payload, "application/json")
	if w.Code != 200 {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	var created struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)
	var col Collection
	db.First(&col, created.ID)
	if col.OwnerID != a.ID || col.ID == legacy.ID {
		t.Fatal("ownership spoof")
	}
	for _, token := range []string{tb, admin} {
		w = memberReq(r, "GET", "/collections", token, "", "")
		if strings.Contains(w.Body.String(), "Alice private") {
			t.Fatal("list leaked")
		}
		for _, item := range []struct{ method, suffix, body string }{{"GET", "/songs", ""}, {"POST", "/songs", `{"id":"x","source":"netease"}`}, {"POST", "/songs/batch", `{"songs":[{"id":"x","source":"netease"}]}`}, {"DELETE", "/songs?id=x&source=netease", ""}, {"PUT", "", `{"name":"stolen"}`}, {"DELETE", "", ""}} {
			w = memberReq(r, item.method, fmt.Sprintf("/collections/%d%s", col.ID, item.suffix), token, item.body, "application/json")
			if w.Code != 404 {
				t.Fatalf("cross-account %s %s status=%d", item.method, item.suffix, w.Code)
			}
		}
		w = memberReq(r, "GET", fmt.Sprintf("/collection?id=%d", col.ID), token, "", "")
		if strings.Contains(w.Body.String(), "Alice private") {
			t.Fatal("detail leaked")
		}
	}
	for _, token := range []string{ta, tb, admin} {
		if w = memberReq(r, "POST", "/collections/liked", token, "", ""); w.Code != 200 {
			t.Fatal(w.Body)
		}
	}
	var likes []Collection
	db.Where("system_key IS NOT NULL").Find(&likes)
	if len(likes) != 3 {
		t.Fatal("liked songs shared")
	}
	for _, path := range []string{"/cookies", "/local_music", "/local_music/cover", "/download?source=local&id=secret", "/search?sources=local", "/user_playlists", "/videos/file", "/api/downloads/queue"} {
		if w = memberReq(r, "GET", path, ta, "", ""); w.Code != 403 {
			t.Fatalf("member route %s=%d", path, w.Code)
		}
	}
	if w = memberReq(r, "POST", "/admin/invites", ta, "", ""); w.Code != 403 {
		t.Fatal("member created invitation")
	}
	if w = memberReq(r, "GET", "/settings", ta, "", ""); w.Code != 200 || strings.Contains(w.Body.String(), "webdavPassword") {
		t.Fatal("settings exposed")
	}
	if w = memberReq(r, "GET", "/search?sources=netease", ta, "", ""); w.Code != 200 {
		t.Fatal("online music unavailable")
	}
	db.Model(&a).Update("disabled", true)
	if w = memberReq(r, "GET", "/collections", ta, "", ""); w.Code != 401 {
		t.Fatal("disabled session accepted")
	}
	if w = memberReq(r, "GET", "/collections", admin, "", ""); !strings.Contains(w.Body.String(), legacy.Name) {
		t.Fatal("legacy collection lost")
	}
}

func TestMemberArtworkRejectsInternalAddresses(t *testing.T) {
	for _, raw := range []string{"http://127.0.0.1/", "http://[::1]/", "http://10.0.0.1/", "http://169.254.169.254/", "http://100.100.100.200/", "file:///etc/passwd", "https://user:pass@example.com/", "http://example.com:37910/"} {
		if publicImageURL(raw) == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	for _, raw := range []string{"127.0.0.1", "::1", "10.0.0.1", "192.168.0.1", "169.254.169.254", "100.100.100.200", "fc00::1"} {
		if publicMediaIP(net.ParseIP(raw)) {
			t.Errorf("accepted IP %s", raw)
		}
	}
	if !publicMediaIP(net.IP{8, 8, 8, 8}) {
		t.Fatal("public IPv4 rejected")
	}
	_, _, err := fetchMemberImage(context.Background(), "http://localhost/")
	if err == nil {
		t.Fatal("resolved loopback accepted")
	}
}
func TestMemberFormsRejectCrossOrigin(t *testing.T) {
	r, _ := memberTestSetup(t)
	for _, path := range []string{"/login", "/register"} {
		req := httptest.NewRequest("POST", "https://music.example/music"+path, strings.NewReader("username=test"))
		req.Header.Set("Origin", "https://evil.example")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatalf("cross-origin %s accepted: %d", path, w.Code)
		}
	}
	for _, name := range []string{"auth.html", "account.html"} {
		data, err := templateFS.ReadFile("templates/pages/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), `name="referrer" content="same-origin"`) {
			t.Fatalf("%s must preserve same-origin form Origin", name)
		}
	}
}

func TestMemberIdentityAndPasswordRotation(t *testing.T) {
	r, _ := memberTestSetup(t)
	m, token := testMember(t, "own_identity")
	w := memberReq(r, "GET", "/account", token, "", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "own_identity") || strings.Contains(w.Body.String(), "你好，owner") {
		t.Fatalf("wrong account identity: %s", w.Body)
	}
	body := url.Values{"current_password": {"test-password-123"}, "password": {"replacement-test-123"}, "password_confirm": {"replacement-test-123"}}
	w = memberReq(r, "POST", "/account/password", token, body.Encode(), "application/x-www-form-urlencoded")
	if w.Code != 303 {
		t.Fatalf("password update failed: %d %s", w.Code, w.Body)
	}
	w = memberReq(r, "GET", "/collections", token, "", "")
	if w.Code != 401 {
		t.Fatal("old session survived password change")
	}
	updated, err := memberAccount(m.Username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("replacement-test-123")) != nil {
		t.Fatal("password not updated")
	}
}
