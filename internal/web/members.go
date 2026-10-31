package web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guohuiyuan/go-music-dl/core"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// The original administrator remains in WebAuthSettings. Owner 0 preserves all
// existing collections; members never receive administrator settings or files.
type WebMember struct {
	ID            uint   `gorm:"primaryKey"`
	Username      string `gorm:"uniqueIndex;not null"`
	PasswordHash  string `json:"-"`
	SessionSecret string `json:"-"`
	Disabled      bool
	CreatedAt     time.Time
}
type WebInvitation struct {
	ID        uint   `gorm:"primaryKey"`
	TokenHash string `gorm:"uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time
	UsedBy    uint
	Revoked   bool
	CreatedAt time.Time
}

func (m WebMember) auth() core.WebAuthSettings {
	return core.WebAuthSettings{Username: m.Username, PasswordHash: m.PasswordHash, SessionSecret: m.SessionSecret}
}
func memberAccount(username string) (WebMember, error) {
	var m WebMember
	if db == nil {
		return m, errors.New("account database unavailable")
	}
	err := db.Where("username = ? AND disabled = ?", strings.ToLower(strings.TrimSpace(username)), false).First(&m).Error
	return m, err
}
func authenticateAccount(c *gin.Context, admin core.WebAuthSettings, value string) bool {
	if validateSessionValue(admin, value, time.Now()) {
		c.Set("AuthUsername", admin.Username)
		c.Set("AuthOwnerID", uint(0))
		c.Set("AuthAdmin", true)
		return true
	}
	if !c.GetBool("PrivateMode") || len(value) > 4096 {
		return false
	}
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	var p sessionPayload
	if json.Unmarshal(raw, &p) != nil {
		return false
	}
	m, err := memberAccount(p.Username)
	if err != nil || !validateSessionValue(m.auth(), value, time.Now()) {
		return false
	}
	c.Set("AuthUsername", m.Username)
	c.Set("AuthOwnerID", m.ID)
	c.Set("AuthAdmin", false)
	return true
}
func isMember(c *gin.Context) bool             { return c.GetBool("PrivateMode") && !c.GetBool("AuthAdmin") }
func collectionOwner(c *gin.Context) uint      { return c.GetUint("AuthOwnerID") }
func ownedCollections(c *gin.Context) *gorm.DB { return db.Where("owner_id = ?", collectionOwner(c)) }
func loadOwnedCollection(c *gin.Context, id string) (*Collection, error) {
	n, err := strconv.ParseUint(id, 10, 32)
	if err != nil || n == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var col Collection
	err = ownedCollections(c).First(&col, uint(n)).Error
	return &col, err
}
func accountStorageKey(c *gin.Context) string {
	if !c.GetBool("PrivateMode") {
		return ""
	}
	// Keep the administrator's existing local playback history intact.
	if c.GetBool("AuthAdmin") {
		return ""
	}
	return fmt.Sprintf(":member:%d", collectionOwner(c))
}
func formSameOrigin(c *gin.Context) bool {
	origin := c.GetHeader("Origin")
	if origin != "" {
		u, e := url.Parse(origin)
		return e == nil && u.Host == c.Request.Host && (u.Scheme == "https" || u.Scheme == "http")
	}
	return c.GetHeader("Sec-Fetch-Site") != "cross-site"
}
func tokenDigest(s string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(s)))
	return hex.EncodeToString(h[:])
}

// Never reuse a previous code, even after expiry, so an old shared code cannot
// accidentally become valid again. The unique index also protects concurrent creation.
func createInvitation() (string, error) {
	for attempt := 0; attempt < 32; attempt++ {
		n, err := rand.Int(rand.Reader, big.NewInt(900000))
		if err != nil {
			return "", err
		}
		code := strconv.FormatInt(n.Int64()+100000, 10)
		inv := WebInvitation{TokenHash: tokenDigest(code), ExpiresAt: time.Now().Add(7 * 24 * time.Hour)}
		result := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "token_hash"}}, DoNothing: true}).Create(&inv)
		if result.Error != nil {
			return "", result.Error
		}
		if result.RowsAffected == 1 {
			return code, nil
		}
	}
	return "", errors.New("邀请码暂时无法生成，请稍后再试")
}

var memberNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{2,31}$`)

func registerMember(code, username, password string) (WebMember, error) {
	var member WebMember
	username = strings.ToLower(strings.TrimSpace(username))
	if !memberNamePattern.MatchString(username) || len(password) < 10 || len(password) > 72 {
		return member, errors.New("用户名需为 3–32 位英文字母、数字或下划线；密码需为 10–72 字节")
	}
	admin, err := core.GetWebAuthSettings()
	if err != nil || !authConfigured(admin) {
		return member, errors.New("管理员尚未完成初始化")
	}
	if strings.EqualFold(username, admin.Username) {
		return member, errors.New("该用户名不可用")
	}
	if len(code) > 128 {
		return member, errors.New("邀请码无效或已使用")
	}
	var inv WebInvitation
	if err = db.Where("token_hash = ? AND used_by = 0 AND revoked = ? AND expires_at > ?", tokenDigest(code), false, time.Now()).First(&inv).Error; err != nil {
		return member, errors.New("邀请码无效、已使用或已过期")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return member, err
	}
	secret, err := randomToken(32)
	if err != nil {
		return member, err
	}
	member = WebMember{Username: username, PasswordHash: string(hash), SessionSecret: secret}
	err = db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&WebMember{}).Count(&count).Error; err != nil {
			return err
		}
		if count >= 100 {
			return errors.New("当前邀请名额已满，请联系管理员")
		}
		// Claim the invitation first. The conditional update is the transaction's
		// write lock; a code cannot create two accounts in concurrent requests.
		result := tx.Model(&WebInvitation{}).Where("id = ? AND used_by = 0 AND revoked = ? AND expires_at > ?", inv.ID, false, time.Now()).Update("revoked", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("邀请码已使用")
		}
		if err := tx.Create(&member).Error; err != nil {
			return errors.New("该用户名不可用，请换一个")
		}
		return tx.Model(&WebInvitation{}).Where("id = ?", inv.ID).Updates(map[string]interface{}{"used_by": member.ID, "revoked": false}).Error
	})
	return member, err
}
func bindRegistrationRoutes(api *gin.RouterGroup) {
	api.GET("/register", func(c *gin.Context) {
		if !c.GetBool("PrivateMode") {
			c.Status(404)
			return
		}
		renderAuthPage(c, "register", "", "")
	})
	api.POST("/register", func(c *gin.Context) {
		if !c.GetBool("PrivateMode") {
			c.Status(404)
			return
		}
		if !formSameOrigin(c) {
			c.Status(403)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
		key := "register|" + c.ClientIP()
		now := time.Now()
		if _, locked := loginLockedUntil(key, now); locked {
			c.String(429, "尝试过于频繁，请稍后再试")
			return
		}
		username := c.PostForm("username")
		password := c.PostForm("password")
		if password != c.PostForm("password_confirm") {
			renderAuthPage(c, "register", "两次输入的密码不一致", username)
			return
		}
		member, err := registerMember(c.PostForm("invite_code"), username, password)
		if err != nil {
			recordLoginFailure(key, now)
			renderAuthPage(c, "register", err.Error(), username)
			return
		}
		clearLoginFailures(key)
		value, err := createSessionValue(member.auth(), now)
		if err != nil {
			c.String(500, "账号已创建，请返回登录")
			return
		}
		setAuthCookie(c, value)
		c.Redirect(303, RoutePrefix+"/")
	})
}

// Deny by default: ordinary members have music discovery/playback and their own
// collections. Server files, platform cookies, jobs and settings stay admin-only.
func memberAccessGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isMember(c) {
			c.Next()
			return
		}
		p := strings.TrimPrefix(c.FullPath(), RoutePrefix)
		allowed := false
		switch p {
		case "/", "/home_feed", "/recommend", "/playlist_categories", "/category_playlists", "/toplists", "/toplist_songs", "/search", "/playlist", "/album", "/album_jump", "/inspect", "/switch_source", "/download", "/download_lrc", "/download_cover", "/cover_proxy", "/lyric", "/my_collections", "/collection", "/settings", "/account":
			allowed = c.Request.Method == "GET"
		}
		if strings.HasPrefix(p, "/collections") && !strings.Contains(p, "local_music") {
			allowed = true
		}
		if p == "/account/password" && c.Request.Method == "POST" {
			allowed = true
		}
		if !allowed {
			c.AbortWithStatusJSON(403, gin.H{"error": "此功能仅限管理员。你可以搜索、播放并管理自己的歌单。"})
			return
		}
		for _, k := range []string{"source", "sources", "current", "target"} {
			for _, v := range c.QueryArray(k) {
				if isLocalMusicSource(v) {
					c.AbortWithStatusJSON(403, gin.H{"error": "服务器本地文件仅限管理员访问"})
					return
				}
			}
		}
		if c.Query("type") == "local_music" || wantsSaveLocal(c) {
			c.AbortWithStatusJSON(403, gin.H{"error": "服务器本地文件仅限管理员访问"})
			return
		}
		if p == "/settings" {
			c.JSON(200, gin.H{"downloadToLocal": false, "embedDownload": false, "autoCacheOnPlay": false, "autoCheckUpdate": false, "disableFloatingLyrics": true, "autoSwitchInvalidSources": true, "webPageSize": 50})
			c.Abort()
			return
		}
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" && !formSameOrigin(c) {
			c.AbortWithStatus(403)
			return
		}
		c.Next()
	}
}
func bindAccountRoutes(api *gin.RouterGroup) {
	api.GET("/account", func(c *gin.Context) { renderAccount(c, "", "") })
	api.POST("/account/password", func(c *gin.Context) {
		if !formSameOrigin(c) {
			c.Status(403)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
		if !isMember(c) {
			renderAccount(c, "管理员密码请由服务器维护者处理。", "")
			return
		}
		m, err := memberAccount(c.GetString("AuthUsername"))
		if err != nil {
			c.Status(401)
			return
		}
		attemptKey := "password|" + m.Username
		if _, locked := loginLockedUntil(attemptKey, time.Now()); locked {
			renderAccount(c, "尝试过于频繁，请稍后再试", "")
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(m.PasswordHash), []byte(c.PostForm("current_password"))) != nil {
			recordLoginFailure(attemptKey, time.Now())
			renderAccount(c, "当前密码不正确", "")
			return
		}
		clearLoginFailures(attemptKey)
		password := c.PostForm("password")
		if len(password) < 10 || len(password) > 72 || password != c.PostForm("password_confirm") {
			renderAccount(c, "新密码需为 10–72 字节，且两次输入一致", "")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			c.Status(500)
			return
		}
		secret, err := randomToken(32)
		if err != nil {
			c.Status(500)
			return
		}
		if db.Model(&m).Updates(map[string]interface{}{"password_hash": string(hash), "session_secret": secret}).Error != nil {
			c.Status(500)
			return
		}
		clearAuthCookie(c)
		c.Redirect(303, RoutePrefix+"/login")
	})
	admin := api.Group("/admin")
	admin.Use(func(c *gin.Context) {
		if !c.GetBool("AuthAdmin") {
			c.AbortWithStatus(403)
		}
	})
	admin.POST("/invites", func(c *gin.Context) {
		if !formSameOrigin(c) {
			c.Status(403)
			return
		}
		var count int64
		db.Model(&WebInvitation{}).Where("used_by = 0 AND revoked = ? AND expires_at > ?", false, time.Now()).Count(&count)
		if count >= 100 {
			renderAccount(c, "未使用邀请码已达 100 个，请先撤销旧邀请码", "")
			return
		}
		code, err := createInvitation()
		if err != nil {
			c.Status(500)
			return
		}
		renderAccount(c, "邀请码已生成：7 天内有效，仅可注册一个账号。", code)
	})
	admin.POST("/invites/:id/revoke", func(c *gin.Context) {
		if !formSameOrigin(c) {
			c.Status(403)
			return
		}
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.Status(400)
			return
		}
		db.Model(&WebInvitation{}).Where("id = ? AND used_by = 0", id).Update("revoked", true)
		c.Redirect(303, RoutePrefix+"/account")
	})
	admin.POST("/members/:id/toggle", func(c *gin.Context) {
		if !formSameOrigin(c) {
			c.Status(403)
			return
		}
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.Status(400)
			return
		}
		var m WebMember
		if db.First(&m, uint(id)).Error != nil {
			c.Status(404)
			return
		}
		secret, err := randomToken(32)
		if err != nil {
			c.Status(500)
			return
		}
		if db.Model(&m).Updates(map[string]interface{}{"disabled": !m.Disabled, "session_secret": secret}).Error != nil {
			c.Status(500)
			return
		}
		c.Redirect(303, RoutePrefix+"/account")
	})
}
func renderAccount(c *gin.Context, message, code string) {
	var members []WebMember
	var invites []WebInvitation
	if c.GetBool("AuthAdmin") {
		db.Select("id", "username", "disabled", "created_at").Order("id DESC").Limit(100).Find(&members)
		db.Select("id", "expires_at", "used_by", "revoked", "created_at").Order("id DESC").Limit(100).Find(&invites)
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "same-origin")
	c.HTML(200, "account.html", gin.H{"Root": RoutePrefix, "Username": c.GetString("AuthUsername"), "Admin": c.GetBool("AuthAdmin"), "Message": message, "Code": code, "Members": members, "Invites": invites})
}
