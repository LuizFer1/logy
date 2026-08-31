package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Release is a GitHub Releases API latest-release payload (fields we need).
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset is a release asset entry.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Getter fetches a URL body.
type Getter interface {
	Get(url string) (body []byte, err error)
}

// HTTPGetter is a Getter backed by net/http.
type HTTPGetter struct {
	Client *http.Client // default timeout 60s
	Token  string       // optional; from GH_TOKEN or GITHUB_TOKEN
}

// Get performs an HTTP GET with GitHub Accept header and optional Bearer token.
func (h HTTPGetter) Get(url string) ([]byte, error) {
	client := h.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	token := h.Token
	if token == "" {
		token = os.Getenv("GH_TOKEN")
		if token == "" {
			token = os.Getenv("GITHUB_TOKEN")
		}
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return body, nil
}

// RepoSlug returns LOGY_GITHUB_REPO or the default "LuizFer1/logy".
func RepoSlug() string {
	if s := strings.TrimSpace(os.Getenv("LOGY_GITHUB_REPO")); s != "" {
		return s
	}
	return "LuizFer1/logy"
}

// LatestRelease fetches https://api.github.com/repos/{repo}/releases/latest.
func LatestRelease(g Getter, repo string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	body, err := g.Get(url)
	if err != nil {
		return nil, err
	}
	var rel Release
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("release missing tag_name")
	}
	return &rel, nil
}

// FindAssetURL returns the browser download URL for the named asset.
func FindAssetURL(rel *Release, name string) (string, error) {
	if rel == nil {
		return "", fmt.Errorf("nil release")
	}
	for _, a := range rel.Assets {
		if a.Name == name {
			if a.BrowserDownloadURL == "" {
				return "", fmt.Errorf("asset %q has empty download URL", name)
			}
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("asset %q not found in release %s", name, rel.TagName)
}

// FindChecksumURL returns the download URL for checksums.txt.
func FindChecksumURL(rel *Release) (string, error) {
	return FindAssetURL(rel, "checksums.txt")
}
