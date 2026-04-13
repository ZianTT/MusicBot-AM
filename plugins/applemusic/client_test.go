package applemusic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientSearchUsesBearerToken(t *testing.T) {
	var authHeader string
	var rawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		rawQuery = r.URL.RawQuery
		if r.URL.Path != "/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": map[string]any{
				"songs": map[string]any{
					"data": []map[string]any{
						{
							"id":   "1480785411",
							"type": "songs",
							"attributes": map[string]any{
								"name":             "記憶の系譜",
								"artistName":       "IOSYS",
								"albumName":        "miko BEST Toho of IOSYS",
								"durationInMillis": 374605,
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", 5*time.Second, nil)
	resp, err := client.Search(context.Background(), "IOSYS", "song", 1)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if authHeader != "Bearer test-token" {
		t.Fatalf("Authorization header = %q, want %q", authHeader, "Bearer test-token")
	}
	if rawQuery != "limit=1&query=IOSYS&type=song" {
		t.Fatalf("raw query = %q", rawQuery)
	}
	if len(resp.Results.Songs.Data) != 1 || resp.Results.Songs.Data[0].ID != "1480785411" {
		t.Fatalf("unexpected search response: %+v", resp.Results.Songs.Data)
	}
}
