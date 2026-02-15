package misskey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/soli0222/diary-cli/internal/models"
)

// Client is a Misskey API client
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a new Misskey client
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetMe returns the authenticated user's information
func (c *Client) GetMe() (*models.MeDetailed, error) {
	resp, err := c.post("/api/i", nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var me models.MeDetailed
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &me, nil
}

// GetUserNotesRequest represents the request parameters for users/notes
type GetUserNotesRequest struct {
	UserID           string `json:"userId"`
	WithReplies      bool   `json:"withReplies"`
	WithRenotes      bool   `json:"withRenotes"`
	WithChannelNotes bool   `json:"withChannelNotes"`
	Limit            int    `json:"limit"`
	SinceDate        *int64 `json:"sinceDate,omitempty"`
	UntilDate        *int64 `json:"untilDate,omitempty"`
	SinceID          string `json:"sinceId,omitempty"`
	UntilID          string `json:"untilId,omitempty"`
}

// GetUserNotes fetches notes for a specific user
func (c *Client) GetUserNotes(req GetUserNotesRequest) ([]models.Note, error) {
	resp, err := c.post("/api/users/notes", req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var notes []models.Note
	if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return notes, nil
}

// GetNotesForDay fetches all notes for a specific day
func (c *Client) GetNotesForDay(userID string, date time.Time, includeRenotes bool) ([]models.Note, error) {
	// Get start and end of day in the local timezone
	loc := date.Location()
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)

	sinceMs := startOfDay.UnixMilli()
	untilMs := endOfDay.UnixMilli()

	var allNotes []models.Note
	var lastID string

	for {
		req := GetUserNotesRequest{
			UserID:           userID,
			WithReplies:      true,
			WithRenotes:      includeRenotes,
			WithChannelNotes: true,
			Limit:            100,
			SinceDate:        &sinceMs,
			UntilDate:        &untilMs,
		}

		if lastID != "" {
			req.SinceID = lastID
		}

		notes, err := c.GetUserNotes(req)
		if err != nil {
			return nil, err
		}

		if len(notes) == 0 {
			break
		}

		allNotes = append(allNotes, notes...)

		// If we got fewer than limit, we've got all notes
		if len(notes) < 100 {
			break
		}

		// Use the last note's ID for pagination
		lastID = notes[len(notes)-1].ID
	}

	return allNotes, nil
}

// GetNotesForTimeRange fetches all notes for a specific time range
func (c *Client) GetNotesForTimeRange(userID string, startTime, endTime time.Time, includeRenotes bool) ([]models.Note, error) {
	sinceMs := startTime.UnixMilli()
	untilMs := endTime.UnixMilli()

	var allNotes []models.Note
	var lastID string

	for {
		req := GetUserNotesRequest{
			UserID:           userID,
			WithReplies:      true,
			WithRenotes:      includeRenotes,
			WithChannelNotes: true,
			Limit:            100,
			SinceDate:        &sinceMs,
			UntilDate:        &untilMs,
		}

		if lastID != "" {
			req.SinceID = lastID
		}

		notes, err := c.GetUserNotes(req)
		if err != nil {
			return nil, err
		}

		if len(notes) == 0 {
			break
		}

		allNotes = append(allNotes, notes...)

		// If we got fewer than limit, we've got all notes
		if len(notes) < 100 {
			break
		}

		// Use the last note's ID for pagination
		lastID = notes[len(notes)-1].ID
	}

	return allNotes, nil
}

// post makes a POST request to the Misskey API
func (c *Client) post(endpoint string, body interface{}) (*http.Response, error) {
	var jsonBody []byte
	var err error

	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	} else {
		jsonBody = []byte("{}")
	}

	req, err := http.NewRequest("POST", c.BaseURL+endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	return c.HTTPClient.Do(req)
}
