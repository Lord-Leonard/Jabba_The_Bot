package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (c *Client) Search(ctx context.Context, query string, params string) ([]SearchResult, error) {
	if query == "" {
		return nil, ErrEmptyQuery
	}

	payload := YoutubeSearchPayload{
		Query: query,
	}
	payload.Context.Client.ClientName = c.clientName
	payload.Context.Client.ClientVersion = c.clientVersion
	payload.Context.Client.Hl = "de"
	payload.Params = params

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("music: error marshalling search payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.searhURL(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("music: error building search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("music: error executing search request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("music: error reading search response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, APIError{Status: resp.StatusCode, Body: string(raw)}
	}

	result, err := parseSearchResult(raw)
	if err != nil {
		return nil, fmt.Errorf("music: error parsing search response: %w", err)
	}
	if len(result) == 0 {
		return nil, ErrNoResults
	}

	return result, nil

}
