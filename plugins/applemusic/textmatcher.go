package applemusic

import (
	"regexp"
	"strings"
)

type TextMatcher struct{}

var appleMusicURLPattern = regexp.MustCompile(`https?://[^\s]+`)

func NewTextMatcher() *TextMatcher { return &TextMatcher{} }

func (m *TextMatcher) MatchText(text string) (trackID string, matched bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	if prefix, value := parseAppleMusicPrefix(text); prefix != "" && isNumericAppleMusicID(value) {
		return value, true
	}
	if urlStr := extractAppleMusicURL(text); urlStr != "" {
		if id, ok := NewURLMatcher().MatchURL(urlStr); ok {
			return id, true
		}
	}
	if isNumericAppleMusicID(text) {
		return text, true
	}
	return "", false
}

func parseAppleMusicPrefix(text string) (string, string) {
	parts := strings.SplitN(text, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	prefix := strings.ToLower(strings.TrimSpace(parts[0]))
	value := strings.TrimSpace(parts[1])
	switch prefix {
	case "applemusic", "apple", "am":
		return prefix, value
	default:
		return "", ""
	}
}

func extractAppleMusicURL(text string) string {
	match := appleMusicURLPattern.FindString(text)
	match = strings.TrimRight(match, ".,!?)]}>")
	return strings.TrimSpace(match)
}
