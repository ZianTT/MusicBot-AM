package applemusic

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/liuran001/MusicBot-Go/bot/platform"
)

type AppleMusicPlatform struct {
	client *Client
}

func NewPlatform(client *Client) *AppleMusicPlatform { return &AppleMusicPlatform{client: client} }

func (a *AppleMusicPlatform) Name() string { return "applemusic" }

func (a *AppleMusicPlatform) SupportsDownload() bool { return true }

func (a *AppleMusicPlatform) SupportsSearch() bool { return true }

func (a *AppleMusicPlatform) SupportsLyrics() bool { return true }

func (a *AppleMusicPlatform) SupportsRecognition() bool { return false }

func (a *AppleMusicPlatform) Capabilities() platform.Capabilities {
	return platform.Capabilities{
		Download: true,
		Search:   true,
		Lyrics:   true,
		HiRes:    true,
	}
}

func (a *AppleMusicPlatform) Metadata() platform.Meta {
	return platform.Meta{
		Name:          "applemusic",
		DisplayName:   "Apple-Music",
		Emoji:         "🍎",
		Aliases:       []string{"applemusic", "apple", "am", "苹果音乐", "apple music"},
		AllowGroupURL: true,
	}
}

func (a *AppleMusicPlatform) GetDownloadInfo(ctx context.Context, trackID string, quality platform.Quality) (*platform.DownloadInfo, error) {
	if a == nil || a.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "track", trackID)
	}
	playback, err := a.client.GetPlaybackInfo(ctx, trackID)
	if err != nil {
		return nil, err
	}
	downloadURL := buildAppleMusicDownloadURL(a.client.baseURL, playback.PlaybackURL)
	if strings.TrimSpace(downloadURL) == "" {
		return nil, platform.NewUnavailableError("applemusic", "track", trackID)
	}
	resolvedQuality := resolveAppleMusicQuality(playback.Codec, quality)
	info := &platform.DownloadInfo{
		URL:     downloadURL,
		Size:    playback.Size,
		Format:  "m4a",
		Bitrate: resolvedQuality.Bitrate(),
		Quality: resolvedQuality,
		Headers: map[string]string{
			"Authorization": "Bearer " + a.client.Token(),
			"Accept":        "audio/*,*/*",
		},
	}
	return info, nil
}

func (a *AppleMusicPlatform) Search(ctx context.Context, query string, limit int) ([]platform.Track, error) {
	if a == nil || a.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "search", "")
	}
	return a.client.SearchSongs(ctx, query, limit)
}

func (a *AppleMusicPlatform) GetLyrics(ctx context.Context, trackID string) (*platform.Lyrics, error) {
	if a == nil || a.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "lyrics", trackID)
	}
	lyric, err := a.client.GetLyrics(ctx, trackID)
	if err != nil {
		return nil, err
	}
	return &platform.Lyrics{
		Plain:       lyric,
		Timestamped: platform.ParseLRCTimestampedLines(lyric),
	}, nil
}

func (a *AppleMusicPlatform) RecognizeAudio(ctx context.Context, audioData io.Reader) (*platform.Track, error) {
	_ = ctx
	_ = audioData
	return nil, platform.NewUnsupportedError("applemusic", "audio recognition")
}

func (a *AppleMusicPlatform) GetTrack(ctx context.Context, trackID string) (*platform.Track, error) {
	if a == nil || a.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "track", trackID)
	}
	return a.client.GetSong(ctx, trackID)
}

func (a *AppleMusicPlatform) GetArtist(ctx context.Context, artistID string) (*platform.Artist, error) {
	if a == nil || a.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "artist", artistID)
	}
	artist, _, err := a.client.GetArtist(ctx, artistID)
	return artist, err
}

func (a *AppleMusicPlatform) GetArtistDetails(ctx context.Context, artistID string) (*platform.Artist, int, error) {
	if a == nil || a.client == nil {
		return nil, 0, platform.NewUnavailableError("applemusic", "artist", artistID)
	}
	return a.client.GetArtist(ctx, artistID)
}

func (a *AppleMusicPlatform) GetAlbum(ctx context.Context, albumID string) (*platform.Album, error) {
	if a == nil || a.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "album", albumID)
	}
	album, _, err := a.client.GetAlbum(ctx, albumID)
	return album, err
}

func (a *AppleMusicPlatform) GetPlaylist(ctx context.Context, playlistID string) (*platform.Playlist, error) {
	isAlbum, rawID := parseCollectionID(playlistID)
	if !isAlbum {
		return nil, platform.NewUnsupportedError("applemusic", "get playlist")
	}
	if a == nil || a.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "album", rawID)
	}
	album, tracks, err := a.client.GetAlbum(ctx, rawID)
	if err != nil {
		return nil, err
	}
	if album == nil {
		return nil, platform.NewNotFoundError("applemusic", "album", rawID)
	}
	creator := ""
	if len(album.Artists) > 0 {
		names := make([]string, 0, len(album.Artists))
		for _, artist := range album.Artists {
			if strings.TrimSpace(artist.Name) != "" {
				names = append(names, strings.TrimSpace(artist.Name))
			}
		}
		creator = strings.Join(names, "/")
	}
	trackCount := album.TrackCount
	if trackCount <= 0 {
		trackCount = len(tracks)
	}
	return &platform.Playlist{
		ID:          rawID,
		Platform:    "applemusic",
		Title:       firstNonEmptyString(album.Title, rawID),
		Description: album.Description,
		CoverURL:    album.CoverURL,
		Creator:     creator,
		TrackCount:  trackCount,
		Tracks:      tracks,
		URL:         album.URL,
	}, nil
}

func (a *AppleMusicPlatform) MatchURL(rawURL string) (string, bool) {
	return NewURLMatcher().MatchURL(rawURL)
}

func (a *AppleMusicPlatform) MatchPlaylistURL(rawURL string) (string, bool) {
	return NewURLMatcher().MatchPlaylistURL(rawURL)
}

func (a *AppleMusicPlatform) MatchArtistURL(rawURL string) (string, bool) {
	return NewURLMatcher().MatchArtistURL(rawURL)
}

func (a *AppleMusicPlatform) MatchText(text string) (string, bool) {
	return NewTextMatcher().MatchText(text)
}

func (a *AppleMusicPlatform) ShortLinkHosts() []string {
	return []string{"music.apple.com"}
}

func (a *AppleMusicPlatform) SupportedLoginMethods() []string {
	return []string{"cookie", "status", "check"}
}

func (a *AppleMusicPlatform) AccountStatus(ctx context.Context) (platform.AccountStatus, error) {
	status := platform.AccountStatus{
		Platform:        a.Name(),
		DisplayName:     a.Metadata().DisplayName,
		AuthMode:        "cookie",
		CanCheckCookie:  true,
		SupportedLogins: a.SupportedLoginMethods(),
	}
	if a == nil || a.client == nil {
		status.Summary = "- 状态: 插件未初始化"
		return status, nil
	}
	token := strings.TrimSpace(a.client.Token())
	if token == "" {
		status.Summary = "- 状态: 未配置 API Token"
		return status, nil
	}
	status.LoggedIn = true
	status.Summary = a.tokenStatusSummary(ctx)
	if strings.Contains(status.Summary, "失败") {
		status.LoggedIn = false
	}
	return status, nil
}

func (a *AppleMusicPlatform) ImportCookie(ctx context.Context, raw string) (platform.CookieImportResult, error) {
	_ = ctx
	if a == nil || a.client == nil {
		return platform.CookieImportResult{}, fmt.Errorf("applemusic client unavailable")
	}
	token := normalizeBearerToken(raw)
	if token == "" {
		return platform.CookieImportResult{}, fmt.Errorf("api token empty")
	}
	a.client.SetToken(token)
	if err := a.client.PersistToken(token); err != nil {
		return platform.CookieImportResult{}, err
	}
	return platform.CookieImportResult{Updated: true, Message: "Apple Music API Token 已更新"}, nil
}

func (a *AppleMusicPlatform) CheckCookie(ctx context.Context) (platform.CookieCheckResult, error) {
	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	info, err := a.GetDownloadInfo(checkCtx, "1480785411", platform.QualityHiRes)
	if err != nil {
		return platform.CookieCheckResult{OK: false, Message: fmt.Sprintf("下载链路校验失败: %v", err)}, nil
	}
	if info == nil || strings.TrimSpace(info.URL) == "" {
		return platform.CookieCheckResult{OK: false, Message: "下载链接为空"}, nil
	}
	return platform.CookieCheckResult{OK: true, Message: "API 鉴权可用（包含文件下载鉴权）"}, nil
}

func buildAppleMusicDownloadURL(baseURL, playbackPath string) string {
	baseURL = strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	playbackPath = strings.TrimSpace(playbackPath)
	if playbackPath == "" {
		return ""
	}
	if strings.HasPrefix(playbackPath, "http://") || strings.HasPrefix(playbackPath, "https://") {
		return playbackPath
	}
	if baseURL == "" {
		return ""
	}
	if !strings.HasPrefix(playbackPath, "/") {
		playbackPath = "/" + playbackPath
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + playbackPath
	}
	resolved, err := u.Parse(playbackPath)
	if err != nil {
		return baseURL + playbackPath
	}
	return resolved.String()
}

func resolveAppleMusicQuality(codec string, requested platform.Quality) platform.Quality {
	codec = strings.ToLower(strings.TrimSpace(codec))
	if strings.Contains(codec, "alac") {
		if requested == platform.QualityHiRes {
			return platform.QualityHiRes
		}
		return platform.QualityLossless
	}
	switch requested {
	case platform.QualityLossless, platform.QualityHiRes:
		return platform.QualityHigh
	case platform.QualityHigh:
		return platform.QualityHigh
	default:
		return platform.QualityStandard
	}
}
