package applemusic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/liuran001/MusicBot-Go/bot"
	"github.com/liuran001/MusicBot-Go/bot/httpproxy"
	"github.com/liuran001/MusicBot-Go/bot/platform"
)

const (
	defaultAppleMusicStorefront = "us"
	defaultAppleMusicLanguage   = "en-US"
)

var defaultAppleMusicArtistViews = []string{
	"top-songs",
	"latest-release",
	"full-albums",
	"singles",
	"featured-playlists",
	"playlists",
	"similar-artists",
	"top-music-videos",
}

type Client struct {
	httpClient  *http.Client
	baseURL     string
	apiToken    string
	storefront  string
	language    string
	logger      bot.Logger
	persistFunc func(map[string]string) error
}

type appleMusicSearchResponse struct {
	Results struct {
		Songs struct {
			Data []appleMusicSongResource `json:"data"`
		} `json:"songs"`
		Albums struct {
			Data []appleMusicAlbumResource `json:"data"`
		} `json:"albums"`
		Artists struct {
			Data []appleMusicArtistResource `json:"data"`
		} `json:"artists"`
	} `json:"results"`
}

type appleMusicAlbumEnvelope struct {
	Data []appleMusicAlbumResource `json:"data"`
}

type appleMusicSongEnvelope struct {
	Data []appleMusicSongResource `json:"data"`
}

type appleMusicArtistEnvelope struct {
	Data []appleMusicArtistResource `json:"data"`
}

type appleMusicArtistViewEnvelope struct {
	Data []appleMusicResourceRef `json:"data"`
}

type appleMusicResourceRef struct {
	ID         string                    `json:"id"`
	Type       string                    `json:"type"`
	Attributes appleMusicCommonAttribute `json:"attributes"`
}

type appleMusicCommonAttribute struct {
	Name       string `json:"name"`
	ArtistName string `json:"artistName"`
	URL        string `json:"url"`
}

type appleMusicArtwork struct {
	URL string `json:"url"`
}

type appleMusicPreview struct {
	URL string `json:"url"`
}

type appleMusicSongResource struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Attributes appleMusicSongAttribute `json:"attributes"`
	Relations  struct {
		Artists struct {
			Data []appleMusicArtistResource `json:"data"`
		} `json:"artists"`
	} `json:"relationships"`
}

type appleMusicSongAttribute struct {
	Name                string              `json:"name"`
	ArtistName          string              `json:"artistName"`
	AlbumName           string              `json:"albumName"`
	DurationInMillis    int64               `json:"durationInMillis"`
	TrackNumber         int                 `json:"trackNumber"`
	DiscNumber          int                 `json:"discNumber"`
	ISRC                string              `json:"isrc"`
	ReleaseDate         string              `json:"releaseDate"`
	HasLyrics           bool                `json:"hasLyrics"`
	HasTimeSyncedLyrics bool                `json:"hasTimeSyncedLyrics"`
	Artwork             appleMusicArtwork   `json:"artwork"`
	Previews            []appleMusicPreview `json:"previews"`
	PlayParams          struct {
		ID string `json:"id"`
	} `json:"playParams"`
}

type appleMusicAlbumResource struct {
	ID         string                   `json:"id"`
	Type       string                   `json:"type"`
	Attributes appleMusicAlbumAttribute `json:"attributes"`
	Relations  struct {
		Artists struct {
			Data []appleMusicArtistResource `json:"data"`
		} `json:"artists"`
		Tracks struct {
			Data []appleMusicSongResource `json:"data"`
		} `json:"tracks"`
	} `json:"relationships"`
}

type appleMusicAlbumAttribute struct {
	Name        string            `json:"name"`
	ArtistName  string            `json:"artistName"`
	TrackCount  int               `json:"trackCount"`
	ReleaseDate string            `json:"releaseDate"`
	Copyright   string            `json:"copyright"`
	Artwork     appleMusicArtwork `json:"artwork"`
}

type appleMusicArtistResource struct {
	ID         string                    `json:"id"`
	Type       string                    `json:"type"`
	Attributes appleMusicArtistAttribute `json:"attributes"`
}

type appleMusicArtistAttribute struct {
	Name string            `json:"name"`
	URL  string            `json:"url"`
	Icon appleMusicArtwork `json:"artwork"`
}

type appleMusicPlayback struct {
	PlaybackURL string `json:"playbackUrl"`
	Size        int64  `json:"size"`
	Codec       string `json:"codec"`
	AlbumID     string `json:"albumId"`
}

type appleMusicLyricsPayload struct {
	Lyrics string `json:"lyrics"`
}

func NewClient(baseURL, apiToken, storefront, language string, timeout time.Duration, logger bot.Logger) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	baseURL = strings.TrimSpace(baseURL)
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
		apiToken:   normalizeBearerToken(apiToken),
		storefront: normalizeStorefront(storefront),
		language:   normalizeLanguage(language),
		logger:     logger,
	}
}

func (c *Client) SetAPIProxy(cfg httpproxy.Config) error {
	if c == nil {
		return nil
	}
	timeout := 15 * time.Second
	if c.httpClient != nil && c.httpClient.Timeout > 0 {
		timeout = c.httpClient.Timeout
	}
	proxiedClient, err := httpproxy.NewHTTPClient(cfg, timeout)
	if err != nil {
		return err
	}
	if proxiedClient == nil {
		c.httpClient = &http.Client{Timeout: timeout}
		return nil
	}
	c.httpClient = proxiedClient
	return nil
}

func (c *Client) SetToken(raw string) {
	if c == nil {
		return
	}
	c.apiToken = normalizeBearerToken(raw)
}

func (c *Client) Token() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.apiToken)
}

func (c *Client) PersistToken(raw string) error {
	if c == nil {
		return fmt.Errorf("applemusic client unavailable")
	}
	if c.persistFunc == nil {
		return fmt.Errorf("applemusic persist func unavailable")
	}
	return c.persistFunc(map[string]string{"api_token": normalizeBearerToken(raw)})
}

func (c *Client) SearchSongs(ctx context.Context, query string, limit int) ([]platform.Track, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, platform.NewNotFoundError("applemusic", "search", "")
	}
	if limit <= 0 {
		limit = 10
	}
	params := url.Values{}
	params.Set("query", query)
	params.Set("type", "song")
	params.Set("limit", strconv.Itoa(limit))
	body, err := c.getJSON(ctx, "/search", params)
	if err != nil {
		return nil, err
	}
	var resp appleMusicSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("applemusic: decode search response: %w", err)
	}
	tracks := make([]platform.Track, 0, len(resp.Results.Songs.Data))
	for _, item := range resp.Results.Songs.Data {
		track := convertAppleMusicSong(item)
		if strings.TrimSpace(track.ID) == "" {
			continue
		}
		tracks = append(tracks, track)
		if len(tracks) >= limit {
			break
		}
	}
	return tracks, nil
}

func (c *Client) GetSong(ctx context.Context, songID string) (*platform.Track, error) {
	songID = strings.TrimSpace(songID)
	if songID == "" {
		return nil, platform.NewNotFoundError("applemusic", "track", songID)
	}
	body, err := c.getJSON(ctx, "/song/"+url.PathEscape(songID), c.withStorefront(nil, false))
	if err != nil {
		return nil, err
	}
	var resp appleMusicSongEnvelope
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("applemusic: decode song response: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, platform.NewNotFoundError("applemusic", "track", songID)
	}
	track := convertAppleMusicSong(resp.Data[0])
	if strings.TrimSpace(track.ID) == "" {
		return nil, platform.NewNotFoundError("applemusic", "track", songID)
	}
	return &track, nil
}

func (c *Client) GetAlbum(ctx context.Context, albumID string) (*platform.Album, []platform.Track, error) {
	albumID = strings.TrimSpace(albumID)
	if albumID == "" {
		return nil, nil, platform.NewNotFoundError("applemusic", "album", albumID)
	}
	body, err := c.getJSON(ctx, "/album/"+url.PathEscape(albumID), c.withStorefront(nil, false))
	if err != nil {
		return nil, nil, err
	}
	var resp appleMusicAlbumEnvelope
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, fmt.Errorf("applemusic: decode album response: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, nil, platform.NewNotFoundError("applemusic", "album", albumID)
	}
	item := resp.Data[0]
	album := convertAppleMusicAlbum(item)
	if album == nil {
		return nil, nil, platform.NewNotFoundError("applemusic", "album", albumID)
	}
	tracks := make([]platform.Track, 0, len(item.Relations.Tracks.Data))
	for _, song := range item.Relations.Tracks.Data {
		track := convertAppleMusicSong(song)
		if strings.TrimSpace(track.ID) == "" {
			continue
		}
		if track.Album == nil {
			track.Album = album
		}
		tracks = append(tracks, track)
	}
	return album, tracks, nil
}

func (c *Client) GetArtist(ctx context.Context, artistID string) (*platform.Artist, int, error) {
	artistID = strings.TrimSpace(artistID)
	if artistID == "" {
		return nil, 0, platform.NewNotFoundError("applemusic", "artist", artistID)
	}
	params := c.withStorefront(nil, true)
	params.Set("views", strings.Join(defaultAppleMusicArtistViews, ","))
	body, err := c.getJSON(ctx, "/artist/"+url.PathEscape(artistID), params)
	if err != nil {
		return nil, 0, err
	}
	var resp appleMusicArtistEnvelope
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, fmt.Errorf("applemusic: decode artist response: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, 0, platform.NewNotFoundError("applemusic", "artist", artistID)
	}
	artist := convertAppleMusicArtist(resp.Data[0])
	if artist == nil {
		return nil, 0, platform.NewNotFoundError("applemusic", "artist", artistID)
	}
	count, _ := c.fetchArtistTopSongsCount(ctx, artistID)
	return artist, count, nil
}

func (c *Client) GetLyrics(ctx context.Context, songID string) (string, error) {
	songID = strings.TrimSpace(songID)
	if songID == "" {
		return "", platform.NewNotFoundError("applemusic", "lyrics", songID)
	}
	body, err := c.getJSON(ctx, "/lyrics/"+url.PathEscape(songID), c.withStorefront(nil, true))
	if err != nil {
		return "", err
	}
	var resp appleMusicLyricsPayload
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("applemusic: decode lyrics response: %w", err)
	}
	lyrics := strings.TrimSpace(resp.Lyrics)
	if lyrics == "" {
		return "", platform.NewUnavailableError("applemusic", "lyrics", songID)
	}
	return lyrics, nil
}

func (c *Client) GetPlaybackInfo(ctx context.Context, songID string) (*appleMusicPlayback, error) {
	songID = strings.TrimSpace(songID)
	if songID == "" {
		return nil, platform.NewNotFoundError("applemusic", "track", songID)
	}
	body, err := c.getJSON(ctx, "/playback/"+url.PathEscape(songID), c.withStorefront(nil, false))
	if err != nil {
		return nil, err
	}
	var resp appleMusicPlayback
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("applemusic: decode playback response: %w", err)
	}
	if strings.TrimSpace(resp.PlaybackURL) == "" {
		return nil, platform.NewUnavailableError("applemusic", "track", songID)
	}
	return &resp, nil
}

func (c *Client) fetchArtistTopSongsCount(ctx context.Context, artistID string) (int, error) {
	params := c.withStorefront(nil, false)
	params.Set("limit", "1")
	params.Set("offset", "0")
	body, err := c.getJSON(ctx, "/artist/"+url.PathEscape(strings.TrimSpace(artistID))+"/view/top-songs", params)
	if err != nil {
		return 0, err
	}
	var resp appleMusicArtistViewEnvelope
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, err
	}
	return len(resp.Data), nil
}

func (c *Client) getJSON(ctx context.Context, pathValue string, query url.Values) ([]byte, error) {
	if c == nil {
		return nil, platform.NewUnavailableError("applemusic", "api", "")
	}
	if strings.TrimSpace(c.baseURL) == "" {
		return nil, platform.NewUnavailableError("applemusic", "api", "base_url")
	}
	token := strings.TrimSpace(c.apiToken)
	if token == "" {
		return nil, platform.NewAuthRequiredError("applemusic")
	}
	endpoint := c.baseURL + pathValue
	if query != nil && len(query) > 0 {
		if strings.Contains(endpoint, "?") {
			endpoint += "&" + query.Encode()
		} else {
			endpoint += "?" + query.Encode()
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, platform.NewAuthRequiredError("applemusic")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, platform.NewNotFoundError("applemusic", "api", pathValue)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, platform.NewRateLimitedError("applemusic")
	}
	if resp.StatusCode >= 400 {
		return nil, platform.NewUnavailableError("applemusic", "api", pathValue)
	}
	return body, nil
}

func normalizeBearerToken(raw string) string {
	raw = strings.TrimSpace(strings.Trim(raw, "`\"'"))
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "bearer "))
	return raw
}

func normalizeStorefront(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return defaultAppleMusicStorefront
	}
	return raw
}

func normalizeLanguage(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultAppleMusicLanguage
	}
	return raw
}

func (c *Client) withStorefront(params url.Values, includeLanguage bool) url.Values {
	if params == nil {
		params = url.Values{}
	}
	if storefront := normalizeStorefront(c.storefront); storefront != "" {
		params.Set("storefront", storefront)
	}
	if includeLanguage {
		if language := normalizeLanguage(c.language); language != "" {
			params.Set("language", language)
		}
	}
	return params
}

func parseAppleMusicDate(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if tm, err := time.Parse("2006-01-02", value); err == nil {
		utc := tm.UTC()
		return &utc
	}
	if tm, err := time.Parse(time.RFC3339, value); err == nil {
		utc := tm.UTC()
		return &utc
	}
	return nil
}

func parseAppleMusicYear(value string) int {
	date := parseAppleMusicDate(value)
	if date == nil {
		return 0
	}
	return date.Year()
}

func convertAppleMusicSong(item appleMusicSongResource) platform.Track {
	id := strings.TrimSpace(item.ID)
	artists := convertAppleMusicArtists(item.Relations.Artists.Data, item.Attributes.ArtistName)
	albumID := ""
	if strings.TrimSpace(item.Attributes.PlayParams.ID) != "" {
		albumID = strings.TrimSpace(item.Attributes.PlayParams.ID)
	}
	album := &platform.Album{
		ID:       albumID,
		Platform: "applemusic",
		Title:    strings.TrimSpace(item.Attributes.AlbumName),
		Artists:  artists,
		CoverURL: normalizeAppleMusicArtwork(item.Attributes.Artwork.URL),
		URL:      buildAppleMusicAlbumURL(albumID),
		Year:     parseAppleMusicYear(item.Attributes.ReleaseDate),
	}
	if strings.TrimSpace(album.Title) == "" && strings.TrimSpace(album.ID) == "" {
		album = nil
	}
	return platform.Track{
		ID:          id,
		Platform:    "applemusic",
		Title:       strings.TrimSpace(item.Attributes.Name),
		Artists:     artists,
		Album:       album,
		Duration:    time.Duration(item.Attributes.DurationInMillis) * time.Millisecond,
		CoverURL:    normalizeAppleMusicArtwork(item.Attributes.Artwork.URL),
		URL:         buildAppleMusicSongURL(id),
		ISRC:        strings.TrimSpace(item.Attributes.ISRC),
		Year:        parseAppleMusicYear(item.Attributes.ReleaseDate),
		TrackNumber: item.Attributes.TrackNumber,
		DiscNumber:  item.Attributes.DiscNumber,
	}
}

func convertAppleMusicAlbum(item appleMusicAlbumResource) *platform.Album {
	id := strings.TrimSpace(item.ID)
	artists := convertAppleMusicArtists(item.Relations.Artists.Data, item.Attributes.ArtistName)
	album := &platform.Album{
		ID:          id,
		Platform:    "applemusic",
		Title:       strings.TrimSpace(item.Attributes.Name),
		Artists:     artists,
		CoverURL:    normalizeAppleMusicArtwork(item.Attributes.Artwork.URL),
		Description: strings.TrimSpace(item.Attributes.Copyright),
		TrackCount:  item.Attributes.TrackCount,
		URL:         buildAppleMusicAlbumURL(id),
		Year:        parseAppleMusicYear(item.Attributes.ReleaseDate),
	}
	album.ReleaseDate = parseAppleMusicDate(item.Attributes.ReleaseDate)
	return album
}

func convertAppleMusicArtist(item appleMusicArtistResource) *platform.Artist {
	id := strings.TrimSpace(item.ID)
	if id == "" {
		return nil
	}
	return &platform.Artist{
		ID:       id,
		Platform: "applemusic",
		Name:     strings.TrimSpace(item.Attributes.Name),
		AvatarURL: normalizeAppleMusicArtwork(
			item.Attributes.Icon.URL,
		),
		URL: firstNonEmptyString(strings.TrimSpace(item.Attributes.URL), buildAppleMusicArtistURL(id)),
	}
}

func convertAppleMusicArtists(items []appleMusicArtistResource, fallbackName string) []platform.Artist {
	artists := make([]platform.Artist, 0, len(items))
	for _, item := range items {
		artist := convertAppleMusicArtist(item)
		if artist == nil {
			continue
		}
		artists = append(artists, *artist)
	}
	if len(artists) == 0 && strings.TrimSpace(fallbackName) != "" {
		artists = append(artists, platform.Artist{
			Platform: "applemusic",
			Name:     strings.TrimSpace(fallbackName),
		})
	}
	return artists
}

func normalizeAppleMusicArtwork(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.ReplaceAll(raw, "{w}", "1200")
	raw = strings.ReplaceAll(raw, "{h}", "1200")
	return raw
}

func buildAppleMusicSongURL(songID string) string {
	songID = strings.TrimSpace(songID)
	if songID == "" {
		return ""
	}
	return "https://music.apple.com/cn/song/" + songID
}

func buildAppleMusicAlbumURL(albumID string) string {
	albumID = strings.TrimSpace(albumID)
	if albumID == "" {
		return ""
	}
	return "https://music.apple.com/cn/album/" + albumID
}

func buildAppleMusicArtistURL(artistID string) string {
	artistID = strings.TrimSpace(artistID)
	if artistID == "" {
		return ""
	}
	return "https://music.apple.com/cn/artist/" + artistID
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
