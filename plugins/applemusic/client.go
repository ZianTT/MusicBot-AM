package applemusic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/liuran001/MusicBot-Go/bot"
	"github.com/liuran001/MusicBot-Go/bot/httpproxy"
)

type Client struct {
	baseURL string
	token   string
	retry   *retryablehttp.Client
	logger  bot.Logger
}

func NewClient(baseURL, token string, timeout time.Duration, logger bot.Logger) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://am.1641263.xyz"
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 2
	retryClient.RetryWaitMin = 200 * time.Millisecond
	retryClient.RetryWaitMax = 2 * time.Second
	retryClient.Logger = nil
	retryClient.HTTPClient.Timeout = timeout
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:   strings.TrimSpace(token),
		retry:   retryClient,
		logger:  logger,
	}
}

func (c *Client) SetAPIProxy(cfg httpproxy.Config) error {
	if c == nil || c.retry == nil {
		return nil
	}
	timeout := 15 * time.Second
	if c.retry.HTTPClient != nil && c.retry.HTTPClient.Timeout > 0 {
		timeout = c.retry.HTTPClient.Timeout
	}
	httpClient, err := httpproxy.NewHTTPClient(cfg, timeout)
	if err != nil {
		return err
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	c.retry.HTTPClient = httpClient
	return nil
}

func (c *Client) Search(ctx context.Context, keyword, searchType string, limit int) (*AMSearchResponse, error) {
	query := url.Values{}
	query.Set("query", keyword)
	if strings.TrimSpace(searchType) != "" {
		query.Set("type", searchType)
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	var result AMSearchResponse
	if err := c.doJSON(ctx, http.MethodGet, "/search", query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetSong(ctx context.Context, id string) (*AMSongResponse, error) {
	var result AMSongResponse
	if err := c.doJSON(ctx, http.MethodGet, "/song/"+url.PathEscape(strings.TrimSpace(id)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetAlbum(ctx context.Context, id string) (*AMAlbumResponse, error) {
	var result AMAlbumResponse
	if err := c.doJSON(ctx, http.MethodGet, "/album/"+url.PathEscape(strings.TrimSpace(id)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetPlayback(ctx context.Context, id string) (*AMPlaybackResponse, error) {
	var result AMPlaybackResponse
	if err := c.doJSON(ctx, http.MethodGet, "/playback/"+url.PathEscape(strings.TrimSpace(id)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetLyrics(ctx context.Context, id string) (*AMLyricsResponse, error) {
	var result AMLyricsResponse
	if err := c.doJSON(ctx, http.MethodGet, "/lyrics/"+url.PathEscape(strings.TrimSpace(id)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, output any) error {
	if c == nil || c.retry == nil {
		return fmt.Errorf("applemusic client unavailable")
	}
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := retryablehttp.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.retry.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("applemusic unauthorized")
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("applemusic resource not found: %s", path)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("applemusic rate limited")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("applemusic api http %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(output)
}
