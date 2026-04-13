package applemusic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/liuran001/MusicBot-Go/bot/platform"
)

func TestAppleMusicPlatformCapabilities(t *testing.T) {
	p := NewPlatform(nil)
	if p.Name() != "applemusic" {
		t.Fatalf("Name() = %q", p.Name())
	}
	if !p.SupportsDownload() || !p.SupportsSearch() || !p.SupportsLyrics() || p.SupportsRecognition() {
		t.Fatalf("unexpected capability flags")
	}
	meta := p.Metadata()
	if meta.Name != "applemusic" {
		t.Fatalf("Metadata().Name = %q", meta.Name)
	}
}

func TestAppleMusicMatcher(t *testing.T) {
	m := NewURLMatcher()
	if id, ok := m.MatchURL("https://music.apple.com/jp/song/test/1480785411"); !ok || id != "1480785411" {
		t.Fatalf("MatchURL() = (%q,%v)", id, ok)
	}
	if id, ok := m.MatchPlaylistURL("https://music.apple.com/jp/album/test/1480785394"); !ok || id != "album:1480785394" {
		t.Fatalf("MatchPlaylistURL() = (%q,%v)", id, ok)
	}
	if id, ok := m.MatchArtistURL("https://music.apple.com/jp/artist/test/287018328"); !ok || id != "287018328" {
		t.Fatalf("MatchArtistURL() = (%q,%v)", id, ok)
	}
}

func TestAppleMusicClientAuthHeaderAndPlaybackFileAuth(t *testing.T) {
	var seenAuth []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = append(seenAuth, strings.TrimSpace(r.Header.Get("Authorization")))
		switch {
		case r.URL.Path == "/search":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": map[string]any{"songs": map[string]any{"data": []any{}}}})
		case r.URL.Path == "/playback/1480785411":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"playbackUrl": "/cache/albums/1480785394/1480785411.m4a",
				"size":        1234,
				"codec":       "ALAC",
				"albumId":     "1480785394",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token-abc", "jp", "ja", 0, nil)
	plat := NewPlatform(client)

	if _, err := client.SearchSongs(context.Background(), "iosys", 5); err != nil {
		t.Fatalf("SearchSongs() error = %v", err)
	}
	info, err := plat.GetDownloadInfo(context.Background(), "1480785411", platform.QualityHiRes)
	if err != nil {
		t.Fatalf("GetDownloadInfo() error = %v", err)
	}
	if info == nil || strings.TrimSpace(info.URL) == "" {
		t.Fatalf("GetDownloadInfo() returned empty url")
	}
	if got := info.Headers["Authorization"]; got != "Bearer token-abc" {
		t.Fatalf("download header auth = %q", got)
	}
	if len(seenAuth) < 2 {
		t.Fatalf("expected auth on api calls, got %d", len(seenAuth))
	}
	for _, auth := range seenAuth {
		if auth != "Bearer token-abc" {
			t.Fatalf("api auth header = %q", auth)
		}
	}
}

func TestAppleMusicTextMatcher(t *testing.T) {
	m := NewTextMatcher()
	if id, ok := m.MatchText("applemusic:1480785411"); !ok || id != "1480785411" {
		t.Fatalf("MatchText(prefix) = (%q,%v)", id, ok)
	}
	if id, ok := m.MatchText("https://music.apple.com/jp/song/test/1480785411"); !ok || id != "1480785411" {
		t.Fatalf("MatchText(url) = (%q,%v)", id, ok)
	}
}
