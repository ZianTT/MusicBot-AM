package applemusic

import (
	"net/url"
	"regexp"
	"strings"
)

type URLMatcher struct{}

var (
	appleMusicSongPathPattern   = regexp.MustCompile(`^[a-z]{2}/song/[^/]+/(\d+)$`)
	appleMusicAlbumPathPattern  = regexp.MustCompile(`^[a-z]{2}/album/[^/]+/(\d+)$`)
	appleMusicArtistPathPattern = regexp.MustCompile(`^[a-z]{2}/artist/[^/]+/(\d+)$`)
)

func NewURLMatcher() *URLMatcher { return &URLMatcher{} }

func (m *URLMatcher) MatchURL(rawURL string) (trackID string, matched bool) {
	parsed, ok := parseAppleMusicURL(rawURL)
	if !ok {
		return "", false
	}
	pathValue := strings.Trim(parsed.Path, "/")
	if match := appleMusicSongPathPattern.FindStringSubmatch(pathValue); len(match) == 2 {
		return strings.TrimSpace(match[1]), true
	}
	query := parsed.Query()
	if songID := strings.TrimSpace(firstNonEmptyString(query.Get("i"), query.Get("songId"), query.Get("song_id"))); isNumericAppleMusicID(songID) {
		return songID, true
	}
	return "", false
}

func (m *URLMatcher) MatchPlaylistURL(rawURL string) (playlistID string, matched bool) {
	parsed, ok := parseAppleMusicURL(rawURL)
	if !ok {
		return "", false
	}
	pathValue := strings.Trim(parsed.Path, "/")
	if match := appleMusicAlbumPathPattern.FindStringSubmatch(pathValue); len(match) == 2 {
		if encoded := encodeAlbumCollectionID(strings.TrimSpace(match[1])); encoded != "" {
			return encoded, true
		}
	}
	query := parsed.Query()
	if albumID := strings.TrimSpace(firstNonEmptyString(query.Get("album"), query.Get("albumId"), query.Get("album_id"))); isNumericAppleMusicID(albumID) {
		if encoded := encodeAlbumCollectionID(albumID); encoded != "" {
			return encoded, true
		}
	}
	return "", false
}

func (m *URLMatcher) MatchArtistURL(rawURL string) (artistID string, matched bool) {
	parsed, ok := parseAppleMusicURL(rawURL)
	if !ok {
		return "", false
	}
	pathValue := strings.Trim(parsed.Path, "/")
	if match := appleMusicArtistPathPattern.FindStringSubmatch(pathValue); len(match) == 2 {
		return strings.TrimSpace(match[1]), true
	}
	query := parsed.Query()
	if artistID := strings.TrimSpace(firstNonEmptyString(query.Get("artist"), query.Get("artistId"), query.Get("artist_id"))); isNumericAppleMusicID(artistID) {
		return artistID, true
	}
	return "", false
}

func parseAppleMusicURL(rawURL string) (*url.URL, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, false
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host != "music.apple.com" && !strings.HasSuffix(host, ".music.apple.com") {
		return nil, false
	}
	return parsed, true
}

func isNumericAppleMusicID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 6 {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
