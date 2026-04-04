package misskey

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestClientGetMe(t *testing.T) {
	var gotAuth string

	client := NewClient("https://misskey.example", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/i" {
			t.Fatalf("path = %q, want /api/i", r.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{"id":"user-1","username":"soli"}`), nil
	})}
	me, err := client.GetMe()
	if err != nil {
		t.Fatalf("GetMe() error = %v", err)
	}

	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if me.ID != "user-1" || me.Username != "soli" {
		t.Fatalf("me = %#v", me)
	}
}

func TestClientGetUserNotesAPIError(t *testing.T) {
	client := NewClient("https://misskey.example", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, "bad request\n"), nil
	})}
	_, err := client.GetUserNotes(GetUserNotesRequest{UserID: "user-1"})
	if err == nil || err.Error() != "API error: 400 - bad request\n" {
		t.Fatalf("err = %v", err)
	}
}

func TestClientGetNotesForTimeRangePaginates(t *testing.T) {
	loc := time.FixedZone("JST", 9*60*60)
	start := time.Date(2026, 2, 23, 5, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)

	var requests []GetUserNotesRequest
	client := NewClient("https://misskey.example", "secret")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/users/notes" {
			t.Fatalf("path = %q, want /api/users/notes", r.URL.Path)
		}

		var req GetUserNotesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		requests = append(requests, req)

		switch req.SinceID {
		case "":
			body, err := json.Marshal(buildMisskeyNotes(100, "note-", start))
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			return jsonResponse(http.StatusOK, string(body)), nil
		case "note-099":
			body, err := json.Marshal(buildMisskeyNotes(2, "tail-", start.Add(100*time.Minute)))
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			return jsonResponse(http.StatusOK, string(body)), nil
		default:
			t.Fatalf("unexpected SinceID = %q", req.SinceID)
		}
		return nil, nil
	})}
	notes, err := client.GetNotesForTimeRange("user-1", start, end, true)
	if err != nil {
		t.Fatalf("GetNotesForTimeRange() error = %v", err)
	}

	if len(notes) != 102 {
		t.Fatalf("len(notes) = %d, want 102", len(notes))
	}
	if len(requests) != 2 {
		t.Fatalf("len(requests) = %d, want 2", len(requests))
	}
	if requests[0].UserID != "user-1" {
		t.Fatalf("UserID = %q", requests[0].UserID)
	}
	if !requests[0].WithReplies || !requests[0].WithRenotes || !requests[0].WithChannelNotes {
		t.Fatalf("request flags = %#v", requests[0])
	}
	if requests[0].Limit != 100 {
		t.Fatalf("Limit = %d", requests[0].Limit)
	}
	if requests[0].SinceDate == nil || *requests[0].SinceDate != start.UnixMilli() {
		t.Fatalf("SinceDate = %v, want %d", requests[0].SinceDate, start.UnixMilli())
	}
	if requests[0].UntilDate == nil || *requests[0].UntilDate != end.UnixMilli() {
		t.Fatalf("UntilDate = %v, want %d", requests[0].UntilDate, end.UnixMilli())
	}
	if requests[1].SinceID != "note-099" {
		t.Fatalf("second SinceID = %q, want note-099", requests[1].SinceID)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func buildMisskeyNotes(count int, prefix string, start time.Time) []map[string]any {
	notes := make([]map[string]any, 0, count)
	for i := range count {
		notes = append(notes, map[string]any{
			"id":        prefix + formatNoteID(i),
			"createdAt": start.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
			"userId":    "user-1",
		})
	}
	return notes
}

func formatNoteID(i int) string {
	return string([]byte{
		byte('0' + (i/100)%10),
		byte('0' + (i/10)%10),
		byte('0' + i%10),
	})
}
