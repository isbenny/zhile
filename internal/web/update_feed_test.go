package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAndroidAssetSelection(t *testing.T) {
	names := []string{"SHA256SUMS.txt", "zhile-source.zip", "zhile-x86_64.apk", "zhile-universal.apk", "zhile-arm64.apk"}
	for arch, want := range map[string]int{"arm64": 4, "amd64": 2, "arm": 3, "386": 3} {
		if got := preferredAndroidAsset(names, arch); got != want {
			t.Fatalf("%s: %d want %d", arch, got, want)
		}
	}
	if preferredAndroidAsset([]string{"notes.txt", "zhile-x86_64.apk"}, "arm64") != -1 {
		t.Fatal("selected incompatible/non-installable asset")
	}
}

func TestHostedUpdateValidationAndFallback(t *testing.T) {
	var base string
	var mode string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assetURL := base + "/downloads/zhile-universal.apk"
		tag := "v9.9.9"
		if mode == "external" {
			assetURL = "https://other.example/zhile-universal.apk"
		}
		if mode == "version" {
			tag = "not-a-version"
		}
		if mode == "oversize" {
			fmt.Fprint(w, strings.Repeat(" ", 256*1024+1))
			return
		}
		if mode == "offline" {
			http.Error(w, "offline", 503)
			return
		}
		fmt.Fprintf(w, `{"tag_name":%q,"html_url":%q,"assets":[{"name":"zhile-universal.apk","size":10,"browser_download_url":%q}]}`, tag, base+"/", assetURL)
	}))
	defer server.Close()
	base = server.URL
	var githubHits atomic.Int32
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		githubHits.Add(1)
		fmt.Fprint(w, `{"tag_name":"v8.0.0","html_url":"https://github.com/isbenny/zhile/releases"}`)
	}))
	defer github.Close()
	previousFeed, previousGitHub := appUpdateFeedURL, githubAPIBaseURL
	appUpdateFeedURL, githubAPIBaseURL = base+"/update.json", github.URL
	t.Cleanup(func() { appUpdateFeedURL, githubAPIBaseURL = previousFeed, previousGitHub })
	release, hosted, err := fetchConfiguredRelease(context.Background(), "isbenny", "zhile", "android", false, "")
	if err != nil || !hosted || release.TagName != "v9.9.9" || githubHits.Load() != 0 {
		t.Fatalf("valid feed not used: %v %t %+v", err, hosted, release)
	}
	for _, invalid := range []string{"external", "version", "oversize", "offline"} {
		mode = invalid
		if _, err := fetchAppUpdateFeed(context.Background(), appUpdateFeedURL); err == nil {
			t.Fatalf("accepted %s feed", mode)
		}
		release, hosted, err = fetchConfiguredRelease(context.Background(), "isbenny", "zhile", "android", false, "")
		if err != nil || hosted || release.TagName != "v8.0.0" {
			t.Fatalf("fallback failed for %s: %v", mode, err)
		}
	}
	mode = ""
	for _, platform := range []string{"darwin", "linux"} {
		_, hosted, err = fetchConfiguredRelease(context.Background(), "isbenny", "zhile", platform, false, "")
		if err != nil || hosted {
			t.Fatal("desktop incorrectly uses APK feed")
		}
	}
	_, hosted, err = fetchConfiguredRelease(context.Background(), "custom", "repo", "android", false, "")
	if err != nil || hosted {
		t.Fatal("custom repo overridden")
	}
}
