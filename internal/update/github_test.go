package update

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRepoSlugDefault(t *testing.T) {
	t.Setenv("LOGY_GITHUB_REPO", "")
	if got := RepoSlug(); got != "LuizFer1/logy" {
		t.Fatalf("got %q", got)
	}
}

func TestRepoSlugOverride(t *testing.T) {
	t.Setenv("LOGY_GITHUB_REPO", "acme/fork")
	if got := RepoSlug(); got != "acme/fork" {
		t.Fatalf("got %q", got)
	}
}

func TestLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/LuizFer1/logy/releases/latest" {
			t.Errorf("path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("Accept=%q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v1.2.0",
			"assets": [
				{"name": "logy_1.2.0_windows_amd64.zip", "browser_download_url": "https://example.com/a.zip"},
				{"name": "checksums.txt", "browser_download_url": "https://example.com/checksums.txt"}
			]
		}`))
	}))
	defer srv.Close()

	g := HTTPGetter{
		Client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				u := *req.URL
				u.Scheme = "http"
				u.Host = strings.TrimPrefix(srv.URL, "http://")
				req2 := req.Clone(req.Context())
				req2.URL = &u
				req2.Host = u.Host
				return http.DefaultTransport.RoundTrip(req2)
			}),
		},
	}

	rel, err := LatestRelease(g, "LuizFer1/logy")
	if err != nil {
		t.Fatal(err)
	}
	if rel.TagName != "v1.2.0" {
		t.Fatalf("tag %q", rel.TagName)
	}
	url, err := FindAssetURL(rel, "logy_1.2.0_windows_amd64.zip")
	if err != nil || url != "https://example.com/a.zip" {
		t.Fatalf("asset url=%q err=%v", url, err)
	}
	curl, err := FindChecksumURL(rel)
	if err != nil || curl != "https://example.com/checksums.txt" {
		t.Fatalf("checksum url=%q err=%v", curl, err)
	}
}

func TestFindAssetURLMissing(t *testing.T) {
	rel := &Release{TagName: "v1.0.0", Assets: nil}
	if _, err := FindAssetURL(rel, "missing.zip"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := FindChecksumURL(rel); err == nil {
		t.Fatal("expected checksum error")
	}
}

func TestHTTPGetterBearerToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"tag_name":"v0.1.0","assets":[]}`))
	}))
	defer srv.Close()

	g := HTTPGetter{
		Client: srv.Client(),
		Token:  "secret-token",
	}
	body, err := g.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "v0.1.0") {
		t.Fatalf("body %s", body)
	}
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("Authorization=%q", gotAuth)
	}
}

func TestVerifySHA256(t *testing.T) {
	data := []byte("hello")
	sum := sha256.Sum256(data)
	want := hex.EncodeToString(sum[:])
	if got := SHA256Hex(data); got != want {
		t.Fatalf("SHA256Hex=%q want %q", got, want)
	}
	if err := VerifySHA256(data, want); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(data, strings.ToUpper(want)); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(data, "deadbeef"); err == nil {
		t.Fatal("expected mismatch")
	}
}
