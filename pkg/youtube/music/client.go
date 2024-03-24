package music

import "net/http"

type Client struct {
	http          *http.Client
	apiKey        string
	baseURL       string
	clientVersion string
	clientName    string
	userAgent     string
}

func NewClient() *Client {
	return &Client{
		http:          &http.Client{},
		apiKey:        "AIzaSyD_Owy59_EfAHrX36X301Q-897dV9_ayRo",
		baseURL:       "https://music.youtube.com/youtubei/v1",
		clientVersion: "1.20240101.01.00",
		clientName:    "WEB_REMIX",
		userAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	q := req.URL.Query()
	q.Add("key", c.apiKey)
	req.URL.RawQuery = q.Encode()

	return c.http.Do(req)
}

func (c *Client) searhURL() string {
	return c.baseURL + "/search"
}

func (c *Client) nextURL() string {
	return c.baseURL + "/next"
}
