package applemusic

type AMSearchResponse struct {
	Results struct {
		Albums AMResourceList[AMAlbum] `json:"albums"`
		Songs  AMResourceList[AMSong]  `json:"songs"`
	} `json:"results"`
}

type AMSongResponse struct {
	Data []AMSong `json:"data"`
}

type AMAlbumResponse struct {
	Data []AMAlbum `json:"data"`
}

type AMResourceList[T any] struct {
	Data []T `json:"data"`
}

type AMSong struct {
	ID            string           `json:"id"`
	Type          string           `json:"type"`
	Href          string           `json:"href"`
	Attributes    AMSongAttributes `json:"attributes"`
	Relationships struct {
		Artists AMResourceList[AMArtist] `json:"artists"`
	} `json:"relationships"`
}

type AMSongAttributes struct {
	Name                string     `json:"name"`
	ArtistName          string     `json:"artistName"`
	AlbumName           string     `json:"albumName"`
	TrackNumber         int        `json:"trackNumber"`
	DiscNumber          int        `json:"discNumber"`
	DurationInMillis    int        `json:"durationInMillis"`
	ReleaseDate         string     `json:"releaseDate"`
	ISRC                string     `json:"isrc"`
	AudioTraits         []string   `json:"audioTraits"`
	HasLyrics           bool       `json:"hasLyrics"`
	HasTimeSyncedLyrics bool       `json:"hasTimeSyncedLyrics"`
	Artwork             *AMArtwork `json:"artwork"`
}

type AMAlbum struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Href          string            `json:"href"`
	Attributes    AMAlbumAttributes `json:"attributes"`
	Relationships struct {
		Tracks  AMResourceList[AMSong]   `json:"tracks"`
		Artists AMResourceList[AMArtist] `json:"artists"`
	} `json:"relationships"`
}

type AMAlbumAttributes struct {
	Name        string     `json:"name"`
	ArtistName  string     `json:"artistName"`
	TrackCount  int        `json:"trackCount"`
	ReleaseDate string     `json:"releaseDate"`
	Artwork     *AMArtwork `json:"artwork"`
}

type AMArtist struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"attributes"`
}

type AMArtwork struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type AMPlaybackResponse struct {
	PlaybackURL string `json:"playbackUrl"`
	Size        int64  `json:"size"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	ArtistID    string `json:"artistId"`
	Album       string `json:"album"`
	AlbumID     string `json:"albumId"`
	Codec       string `json:"codec"`
}

type AMLyricsResponse struct {
	Lyrics string `json:"lyrics"`
}
