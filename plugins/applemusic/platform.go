package applemusic

import (
	"context"
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/liuran001/MusicBot-Go/bot/platform"
)

type AppleMusicPlatform struct {
	client *Client
}

func NewPlatform(client *Client) *AppleMusicPlatform {
	return &AppleMusicPlatform{client: client}
}

func (p *AppleMusicPlatform) Name() string { return "applemusic" }

func (p *AppleMusicPlatform) SupportsDownload() bool { return true }

func (p *AppleMusicPlatform) SupportsSearch() bool { return true }

func (p *AppleMusicPlatform) SupportsLyrics() bool { return true }

func (p *AppleMusicPlatform) SupportsRecognition() bool { return false }

func (p *AppleMusicPlatform) Capabilities() platform.Capabilities {
	return platform.Capabilities{Download: true, Search: true, Lyrics: true, Recognition: false, HiRes: true}
}

func (p *AppleMusicPlatform) Metadata() platform.Meta {
	return platform.Meta{
		Name:          "applemusic",
		DisplayName:   "Apple Music",
		Emoji:         "🍎",
		Aliases:       []string{"applemusic", "apple", "am"},
		AllowGroupURL: true,
	}
}

func (p *AppleMusicPlatform) GetDownloadInfo(ctx context.Context, trackID string, quality platform.Quality) (*platform.DownloadInfo, error) {
	_ = quality
	if p == nil || p.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "track", trackID)
	}
	playback, err := p.client.GetPlayback(ctx, trackID)
	if err != nil {
		return nil, platform.NewUnavailableError("applemusic", "track", trackID)
	}
	playbackURL := strings.TrimSpace(playback.PlaybackURL)
	if playbackURL == "" {
		return nil, platform.NewUnavailableError("applemusic", "track", trackID)
	}
	if !strings.HasPrefix(playbackURL, "http://") && !strings.HasPrefix(playbackURL, "https://") {
		playbackURL = strings.TrimRight(p.client.baseURL, "/") + "/" + strings.TrimLeft(playbackURL, "/")
	}
	format := strings.TrimPrefix(strings.ToLower(path.Ext(playbackURL)), ".")
	if format == "" {
		format = "m4a"
	}
	return &platform.DownloadInfo{
		URL:     playbackURL,
		Size:    playback.Size,
		Format:  format,
		Quality: mapPlaybackQuality(playback.Codec),
	}, nil
}

func mapPlaybackQuality(codec string) platform.Quality {
	codec = strings.ToLower(strings.TrimSpace(codec))
	if codec == "alac" {
		return platform.QualityLossless
	}
	return platform.QualityHigh
}

func (p *AppleMusicPlatform) Search(ctx context.Context, query string, limit int) ([]platform.Track, error) {
	if p == nil || p.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "search", query)
	}
	res, err := p.client.Search(ctx, query, "song", limit)
	if err != nil {
		return nil, err
	}
	tracks := make([]platform.Track, 0, len(res.Results.Songs.Data))
	for _, song := range res.Results.Songs.Data {
		tracks = append(tracks, p.mapSong(song))
	}
	return tracks, nil
}

func (p *AppleMusicPlatform) GetLyrics(ctx context.Context, trackID string) (*platform.Lyrics, error) {
	if p == nil || p.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "lyrics", trackID)
	}
	res, err := p.client.GetLyrics(ctx, trackID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(res.Lyrics) == "" {
		return nil, platform.NewUnavailableError("applemusic", "lyrics", trackID)
	}
	return &platform.Lyrics{
		Plain:       res.Lyrics,
		Timestamped: platform.ParseLRCTimestampedLines(res.Lyrics),
	}, nil
}

func (p *AppleMusicPlatform) RecognizeAudio(ctx context.Context, audioData io.Reader) (*platform.Track, error) {
	_ = ctx
	_ = audioData
	return nil, platform.NewUnsupportedError("applemusic", "audio recognition")
}

func (p *AppleMusicPlatform) GetTrack(ctx context.Context, trackID string) (*platform.Track, error) {
	if p == nil || p.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "track", trackID)
	}
	res, err := p.client.GetSong(ctx, trackID)
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, platform.NewNotFoundError("applemusic", "track", trackID)
	}
	track := p.mapSong(res.Data[0])
	return &track, nil
}

func (p *AppleMusicPlatform) GetArtist(ctx context.Context, artistID string) (*platform.Artist, error) {
	_ = ctx
	return nil, platform.NewUnsupportedError("applemusic", "artist")
}

func (p *AppleMusicPlatform) GetAlbum(ctx context.Context, albumID string) (*platform.Album, error) {
	if p == nil || p.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "album", albumID)
	}
	res, err := p.client.GetAlbum(ctx, albumID)
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, platform.NewNotFoundError("applemusic", "album", albumID)
	}
	return p.mapAlbum(res.Data[0]), nil
}

func (p *AppleMusicPlatform) GetPlaylist(ctx context.Context, playlistID string) (*platform.Playlist, error) {
	if p == nil || p.client == nil {
		return nil, platform.NewUnavailableError("applemusic", "playlist", playlistID)
	}
	res, err := p.client.GetAlbum(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, platform.NewNotFoundError("applemusic", "playlist", playlistID)
	}
	album := res.Data[0]
	tracks := make([]platform.Track, 0, len(album.Relationships.Tracks.Data))
	for _, song := range album.Relationships.Tracks.Data {
		if strings.TrimSpace(song.Attributes.AlbumName) == "" {
			song.Attributes.AlbumName = album.Attributes.Name
		}
		if song.Attributes.Artwork == nil {
			song.Attributes.Artwork = album.Attributes.Artwork
		}
		tracks = append(tracks, p.mapSong(song))
	}
	return &platform.Playlist{
		ID:         album.ID,
		Platform:   p.Name(),
		Title:      album.Attributes.Name,
		CoverURL:   normalizeArtworkURL(album.Attributes.Artwork),
		Creator:    album.Attributes.ArtistName,
		TrackCount: maxInt(album.Attributes.TrackCount, len(tracks)),
		Tracks:     tracks,
	}, nil
}

func (p *AppleMusicPlatform) MatchURL(rawURL string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", false
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host != "music.apple.com" {
		return "", false
	}
	if id := strings.TrimSpace(u.Query().Get("i")); id != "" {
		return id, true
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if _, err := strconv.ParseInt(parts[i], 10, 64); err == nil {
			return parts[i], true
		}
	}
	return "", false
}

func (p *AppleMusicPlatform) mapSong(song AMSong) platform.Track {
	artists := make([]platform.Artist, 0, len(song.Relationships.Artists.Data))
	for _, artist := range song.Relationships.Artists.Data {
		name := strings.TrimSpace(artist.Attributes.Name)
		if name == "" {
			continue
		}
		artists = append(artists, platform.Artist{
			ID:       artist.ID,
			Platform: p.Name(),
			Name:     name,
			URL:      artist.Attributes.URL,
		})
	}
	if len(artists) == 0 && strings.TrimSpace(song.Attributes.ArtistName) != "" {
		artists = append(artists, platform.Artist{
			Platform: p.Name(),
			Name:     song.Attributes.ArtistName,
		})
	}
	year := 0
	if t, err := time.Parse("2006-01-02", song.Attributes.ReleaseDate); err == nil {
		year = t.Year()
	}
	album := &platform.Album{
		Platform: p.Name(),
		Title:    song.Attributes.AlbumName,
		CoverURL: normalizeArtworkURL(song.Attributes.Artwork),
	}
	return platform.Track{
		ID:          song.ID,
		Platform:    p.Name(),
		Title:       song.Attributes.Name,
		Artists:     artists,
		Album:       album,
		Duration:    time.Duration(song.Attributes.DurationInMillis) * time.Millisecond,
		CoverURL:    normalizeArtworkURL(song.Attributes.Artwork),
		ISRC:        song.Attributes.ISRC,
		Year:        year,
		TrackNumber: song.Attributes.TrackNumber,
		DiscNumber:  song.Attributes.DiscNumber,
	}
}

func (p *AppleMusicPlatform) mapAlbum(album AMAlbum) *platform.Album {
	artists := make([]platform.Artist, 0, len(album.Relationships.Artists.Data))
	for _, artist := range album.Relationships.Artists.Data {
		if strings.TrimSpace(artist.Attributes.Name) == "" {
			continue
		}
		artists = append(artists, platform.Artist{
			ID:       artist.ID,
			Platform: p.Name(),
			Name:     artist.Attributes.Name,
			URL:      artist.Attributes.URL,
		})
	}
	if len(artists) == 0 && strings.TrimSpace(album.Attributes.ArtistName) != "" {
		artists = append(artists, platform.Artist{
			Platform: p.Name(),
			Name:     album.Attributes.ArtistName,
		})
	}
	var releaseDate *time.Time
	year := 0
	if t, err := time.Parse("2006-01-02", album.Attributes.ReleaseDate); err == nil {
		releaseDate = &t
		year = t.Year()
	}
	return &platform.Album{
		ID:          album.ID,
		Platform:    p.Name(),
		Title:       album.Attributes.Name,
		Artists:     artists,
		CoverURL:    normalizeArtworkURL(album.Attributes.Artwork),
		ReleaseDate: releaseDate,
		TrackCount:  album.Attributes.TrackCount,
		Year:        year,
	}
}

func normalizeArtworkURL(artwork *AMArtwork) string {
	if artwork == nil {
		return ""
	}
	cover := strings.TrimSpace(artwork.URL)
	cover = strings.ReplaceAll(cover, "{w}", "600")
	cover = strings.ReplaceAll(cover, "{h}", "600")
	return cover
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
