package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func (c *Client) GetNextSongs(ctx context.Context, videoID string) ([]SearchResult, error) {
	if videoID == "" {
		return nil, errors.New("video ID cannot be empty")
	}

	payload := YoutubeSearchPayload{
		VideoID:                       videoID,
		PlaylistID:                    "RDAMVM" + videoID,
		EnablePersistentPlaylistPanel: true,
		IsAudioOnly:                   true,
	}
	payload.Context.Client.ClientName = "WEB_REMIX"
	payload.Context.Client.ClientVersion = c.clientVersion
	payload.Context.Client.Hl = "de"

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("music: error marshalling search payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.nextURL(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("music: error building next request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("music: error executing next request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("music: error reading next response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, APIError{Status: resp.StatusCode, Body: string(raw)}
	}

	result, err := parseNextResult(raw)
	if err != nil {
		return nil, fmt.Errorf("music: error parsing next response: %w", err)
	}
	if len(result) == 0 {
		return nil, ErrNoResults
	}

	return result, nil
}
