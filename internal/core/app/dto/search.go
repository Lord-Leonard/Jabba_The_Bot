package dto

type SearchResult struct {
	Title       string `json:"title,omitempty"`
	Artist      string `json:"artist,omitempty"`
	VideoID     string `json:"videoId,omitempty"`
	CoverArtURL string `json:"coverArtUrl,omitempty"`
}
