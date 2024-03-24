package music

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

func parseSearchResult(raw []byte) ([]SearchResult, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var searchResponse SearchResponse
	err := json.Unmarshal(raw, &searchResponse)
	if err != nil {
		return nil, fmt.Errorf("music: error parsing next response: %w", err)
	}

	contents := searchResponse.Contents.TabbedSearchResultsRenderer.Tabs[0].TabRenderer.Content.SectionListRenderer.Contents

	results := make([]SearchResult, 0)
	add := func(result SearchResult) {
		if result.VideoID == "" || result.Title == "" {
			return
		}
		results = append(results, result)
	}

	for _, result := range contents {
		card := result.MusicCardShelfRenderer
		if card != nil {
			titleRuns := card.Title.Runs
			if len(titleRuns) == 0 {
				continue
			}

			coverArtURL := ""
			thumbnails := card.Thumbnail.MusicThumbnailRenderer.Thumbnail.Thumbnails
			if len(thumbnails) > 0 {
				coverArtURL = parseThumbnailURL(thumbnails[0].URL)
			}

			add(SearchResult{
				CoverArtUrl: coverArtURL,
				VideoID:     runWatchVideoID(titleRuns[0]),
				Title:       titleRuns[0].Text,
				Artist:      extractArtist(card.Subtitle.Runs),
			})

			for _, content := range card.Contents {
				item := content.MusicResponsiveListItemRenderer
				if len(item.FlexColumns) == 0 {
					continue
				}

				runs := item.FlexColumns[0].MusicResponsiveListItemFlexColumnRenderer.Text.Runs
				if len(runs) == 0 {
					continue
				}

				videoID := runWatchVideoID(runs[0])
				if videoID == "" {
					videoID = item.Overlay.MusicItemThumbnailOverlayRenderer.Content.MusicPlayButtonRenderer.PlayNavigationEndpoint.WatchEndpoint.VideoID
				}

				coverArtURL := ""
				thumbnails := item.Thumbnail.MusicThumbnailRenderer.Thumbnail.Thumbnails
				if len(thumbnails) > 0 {
					coverArtURL = parseThumbnailURL(thumbnails[0].URL)
				}

				add(SearchResult{
					CoverArtUrl: coverArtURL,
					Title:       runs[0].Text,
					VideoID:     videoID,
					Artist:      extractArtist(runs),
				})
			}
		}

		shelf := result.MusicShelfRenderer
		if shelf != nil {
			for _, shelfItem := range shelf.Contents {
				item := shelfItem.MusicResponsiveListItemRenderer
				if len(item.FlexColumns) == 0 {
					continue
				}

				runs := item.FlexColumns[0].MusicResponsiveListItemFlexColumnRenderer.Text.Runs
				if len(runs) == 0 {
					continue
				}
				videoID := runWatchVideoID(runs[0])
				if videoID == "" {
					videoID = item.PlaylistItemData.VideoID
				}
				if videoID == "" {
					videoID = gjson.GetBytes(
						item.Overlay.MusicItemThumbnailOverlayRenderer.Content.MusicPlayButtonRenderer,
						"playNavigationEndpoint.watchEndpoint.videoId",
					).String()
				}

				coverArtURL := ""
				thumbnails := item.Thumbnail.MusicThumbnailRenderer.Thumbnail.Thumbnails
				if len(thumbnails) > 0 {
					coverArtURL = parseThumbnailURL(thumbnails[0].URL)
				}

				add(SearchResult{
					CoverArtUrl: coverArtURL,
					Title:       runs[0].Text,
					VideoID:     videoID,
					Artist:      extractArtist(runs),
				})
			}
		}
	}

	if len(results) == 0 {
		return nil, nil
	}

	return results, nil
}

func extractArtist(runs []Run) string {
	for _, run := range runs {
		if run.NavigationEndpoint == nil {
			continue
		}
		if run.NavigationEndpoint.BrowseEndpoint == nil {
			continue
		}

		PageType := run.NavigationEndpoint.BrowseEndpoint.BrowseEndpointContextSupportedConfigs.BrowseEndpointContextMusicConfig.PageType
		if PageType == "MUSIC_PAGE_TYPE_ARTIST" {
			return run.Text
		}
		if PageType == "MUSIC_PAGE_TYPE_USER_CHANNEL" {
			return run.Text
		}
	}

	return ""
}

func runWatchVideoID(run Run) string {
	if run.NavigationEndpoint == nil {
		return ""
	}
	if run.NavigationEndpoint.WatchEndpoint == nil {
		return ""
	}

	return run.NavigationEndpoint.WatchEndpoint.VideoID
}

func parseNextResult(raw []byte) ([]SearchResult, error) {
	var nextResponse YoutubeNextResponse
	err := json.Unmarshal(raw, &nextResponse)
	if err != nil {
		return nil, fmt.Errorf("music: error parsing next response: %w", err)
	}

	searchResults := make([]SearchResult, 0)
	contents := nextResponse.Contents.SingleColumnMusicWatchNextResultsRenderer.TabbedRenderer.WatchNextTabbedResultsRenderer.Tabs[0].TabRenderer.Content.MusicQueueRenderer.Content.PlaylistPanelRenderer.Contents
	for _, content := range contents {
		searchResult := SearchResult{
			Title:   content.PlaylistPanelVideoRenderer.Title.Runs[0].Text,
			VideoID: content.PlaylistPanelVideoRenderer.VideoID,
		}
		searchResults = append(searchResults, searchResult)
	}
	return searchResults, nil
}

var ytThumb = regexp.MustCompile(`/vi/([^/]+)/`)

func parseThumbnailURL(url string) string {
	if strings.Contains(url, "i.ytimg.com") {
		m := ytThumb.FindStringSubmatch(url)
		if len(m) < 2 {
			return url
		}

		videoID := m[1]
		return fmt.Sprintf("https://i.ytimg.com/vi/%s/%s.jpg", videoID, "hqdefault")
	}

	//if strings.Contains(url, "googleusercontent.com") {
	//	lastEq := strings.LastIndex(url, "=")
	//	if lastEq == -1 {
	//		return url
	//	}
	//
	//	// Ensure it's not part of a query parameter
	//	if strings.Contains(url[lastEq:], "&") {
	//		return url
	//	}
	//
	//	return url[:lastEq] + "=s0-rj"
	//}

	return url

}
