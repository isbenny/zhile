package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var appUpdateFeedURL = "https://music.cnxwkj.com/update.json"
var releaseVersionPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:\.[0-9]+)?$`)

// Only the official Android distribution uses the self-hosted APK channel.
// Custom repository settings and desktop distributions keep their own channel.
func fetchConfiguredRelease(ctx context.Context, owner, repo, platform string, proxyEnabled bool, proxyURL string) (githubRelease, bool, error) {
	if platform == "android" && strings.EqualFold(owner, "isbenny") && strings.EqualFold(repo, "zhile") {
		if release, err := fetchAppUpdateFeed(ctx, appUpdateFeedURL); err == nil {
			return release, true, nil
		}
	}
	release, err := fetchLatestGitHubRelease(ctx, owner, repo, proxyEnabled, proxyURL)
	return release, false, err
}

func fetchAppUpdateFeed(ctx context.Context, endpoint string) (githubRelease, error) {
	var release githubRelease
	base, err := url.Parse(endpoint)
	if err != nil || base.Host == "" || base.User != nil {
		return release, fmt.Errorf("invalid update endpoint")
	}
	sameOrigin := func(raw string) bool {
		u, err := url.Parse(raw)
		return err == nil && u.User == nil && u.Host == base.Host && u.Scheme == base.Scheme && u.Fragment == ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return release, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-cache")
	client := &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 || !sameOrigin(req.URL.String()) {
			return fmt.Errorf("update redirect rejected")
		}
		return nil
	}}
	resp, err := client.Do(req)
	if err != nil {
		return release, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return release, fmt.Errorf("update feed HTTP %d", resp.StatusCode)
	}
	const limit = 256 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(body) > limit {
		return release, fmt.Errorf("cannot read update feed")
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return release, err
	}
	if !releaseVersionPattern.MatchString(release.TagName) || !sameOrigin(release.HTMLURL) || len(release.Assets) == 0 {
		return githubRelease{}, fmt.Errorf("invalid update release")
	}
	seen := map[string]bool{}
	for _, asset := range release.Assets {
		if asset.Name != "zhile-arm64.apk" && asset.Name != "zhile-x86_64.apk" && asset.Name != "zhile-universal.apk" {
			return githubRelease{}, fmt.Errorf("unexpected update asset")
		}
		u, err := url.Parse(asset.BrowserDownloadURL)
		if err != nil || !sameOrigin(asset.BrowserDownloadURL) || u.Path != "/downloads/"+asset.Name || u.RawQuery != "" || asset.Size <= 0 || seen[asset.Name] {
			return githubRelease{}, fmt.Errorf("invalid update asset")
		}
		seen[asset.Name] = true
	}
	// All architectures must have a usable fallback before promoting the feed.
	if !seen["zhile-universal.apk"] {
		return githubRelease{}, fmt.Errorf("missing universal APK")
	}
	return release, nil
}

func preferredAndroidAsset(names []string, arch string) int {
	best, score := -1, 0
	for i, name := range names {
		name = strings.ToLower(name)
		if !strings.HasSuffix(name, ".apk") {
			continue
		}
		candidate := 0
		if strings.Contains(name, "universal") {
			candidate = 100
		}
		aliases := map[string][]string{"arm64": {"arm64", "aarch64"}, "amd64": {"x86_64", "amd64", "x64"}, "386": {"x86.apk", "386"}, "arm": {"armeabi-v7a", "armv7"}}
		for _, alias := range aliases[arch] {
			if strings.Contains(name, alias) {
				candidate = 200
			}
		}
		if candidate > score {
			best, score = i, candidate
		}
	}
	return best
}
