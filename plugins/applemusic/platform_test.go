package applemusic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/liuran001/MusicBot-Go/bot/platform"
)

func TestPlatformGetDownloadInfoBuildsAbsoluteURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/playback/1480785411" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"playbackUrl": "cache/albums/1480785394/1480785411.m4a",
			"size":        int64(77092711),
			"codec":       "ALAC",
		})
	}))
	defer server.Close()

	p := NewPlatform(NewClient(server.URL, "token", 5*time.Second, nil))
	info, err := p.GetDownloadInfo(context.Background(), "1480785411", platform.QualityHigh)
	if err != nil {
		t.Fatalf("GetDownloadInfo() error = %v", err)
	}
	if info.URL != server.URL+"/cache/albums/1480785394/1480785411.m4a" {
		t.Fatalf("GetDownloadInfo() URL = %q", info.URL)
	}
	if info.Format != "m4a" {
		t.Fatalf("GetDownloadInfo() format = %q", info.Format)
	}
	if info.Quality != platform.QualityLossless {
		t.Fatalf("GetDownloadInfo() quality = %q", info.Quality)
	}
	if info.Headers["Authorization"] != "Bearer token" {
		t.Fatalf("GetDownloadInfo() Authorization header = %q", info.Headers["Authorization"])
	}
}

func TestPlatformGetPlaylistMapsAlbumTracks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/album/1480785394" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":   "1480785394",
					"type": "albums",
					"attributes": map[string]any{
						"name":       "miko BEST Toho of IOSYS",
						"artistName": "IOSYS",
						"trackCount": 1,
						"artwork": map[string]any{
							"url": "https://is1-ssl.mzstatic.com/image/thumb/{w}x{h}bb.jpg",
						},
					},
					"relationships": map[string]any{
						"tracks": map[string]any{
							"data": []map[string]any{
								{
									"id":   "1480785411",
									"type": "songs",
									"attributes": map[string]any{
										"name":             "記憶の系譜",
										"artistName":       "IOSYS",
										"albumName":        "miko BEST Toho of IOSYS",
										"durationInMillis": 374605,
										"trackNumber":      17,
										"discNumber":       1,
									},
								},
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	p := NewPlatform(NewClient(server.URL, "token", 5*time.Second, nil))
	playlist, err := p.GetPlaylist(context.Background(), "1480785394")
	if err != nil {
		t.Fatalf("GetPlaylist() error = %v", err)
	}
	if playlist.Platform != "applemusic" || playlist.ID != "1480785394" {
		t.Fatalf("unexpected playlist identity: %+v", playlist)
	}
	if len(playlist.Tracks) != 1 || playlist.Tracks[0].ID != "1480785411" {
		t.Fatalf("unexpected playlist tracks: %+v", playlist.Tracks)
	}
	if playlist.CoverURL != "https://is1-ssl.mzstatic.com/image/thumb/600x600bb.jpg" {
		t.Fatalf("playlist cover url = %q", playlist.CoverURL)
	}
}
