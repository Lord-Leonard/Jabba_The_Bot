package youtube

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
		return nil, fmt.Errorf("music: error parsing search response: %w", err)
	}

	contents := searchResponse.Contents.TabbedSearchResultsRenderer.Tabs[0].TabRenderer.Content.SectionListRenderer.Contents

	results := make([]SearchResult, 0)

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

			appendSearchResult(&results, SearchResult{
				CoverArtUrl: coverArtURL,
				VideoID:     runWatchVideoID(titleRuns[0]),
				Title:       titleRuns[0].Text,
				Artist:      extractArtist(card.Subtitle.Runs),
				Album:       extractAlbum(card.Subtitle.Runs),
			})

			for _, content := range card.Contents {
				item := content.MusicResponsiveListItemRenderer

				titleRuns := flexColumnRuns(item.FlexColumns, 0)
				bylineRuns := flexColumnRuns(item.FlexColumns, 1)
				videoID := ""
				if len(titleRuns) > 0 {
					videoID = runWatchVideoID(titleRuns[0])
				}
				if videoID == "" {
					videoID = item.PlaylistItemData.VideoID
				}
				if videoID == "" {
					videoID = item.Overlay.MusicItemThumbnailOverlayRenderer.Content.MusicPlayButtonRenderer.PlayNavigationEndpoint.WatchEndpoint.VideoID
				}

				itemCoverArtURL := ""
				itemThumbnails := item.Thumbnail.MusicThumbnailRenderer.Thumbnail.Thumbnails
				if len(itemThumbnails) > 0 {
					itemCoverArtURL = parseThumbnailURL(itemThumbnails[0].URL)
				}

				appendSearchResult(&results, parseListItemResult(titleRuns, bylineRuns, videoID, itemCoverArtURL))
			}
		}

		shelf := result.MusicShelfRenderer
		if shelf != nil {
			for _, shelfItem := range shelf.Contents {
				item := shelfItem.MusicResponsiveListItemRenderer
				titleRuns := flexColumnRuns(item.FlexColumns, 0)
				bylineRuns := flexColumnRuns(item.FlexColumns, 1)

				videoID := ""
				if len(titleRuns) > 0 {
					videoID = runWatchVideoID(titleRuns[0])
				}
				if videoID == "" {
					videoID = item.PlaylistItemData.VideoID
				}
				if videoID == "" {
					videoID = gjson.GetBytes(
						item.Overlay.MusicItemThumbnailOverlayRenderer.Content.MusicPlayButtonRenderer,
						"playNavigationEndpoint.watchEndpoint.videoId",
					).String()
				}

				itemCoverArtURL := ""
				itemThumbnails := item.Thumbnail.MusicThumbnailRenderer.Thumbnail.Thumbnails
				if len(itemThumbnails) > 0 {
					itemCoverArtURL = parseThumbnailURL(itemThumbnails[0].URL)
				}

				appendSearchResult(&results, parseListItemResult(titleRuns, bylineRuns, videoID, itemCoverArtURL))
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

func extractAlbum(runs []Run) string {
	for _, run := range runs {
		if run.NavigationEndpoint == nil {
			continue
		}
		if run.NavigationEndpoint.BrowseEndpoint == nil {
			continue
		}

		PageType := run.NavigationEndpoint.BrowseEndpoint.BrowseEndpointContextSupportedConfigs.BrowseEndpointContextMusicConfig.PageType
		if PageType == "MUSIC_PAGE_TYPE_ALBUM" {
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
	if len(raw) == 0 {
		return nil, nil
	}

	var nextResponse YoutubeNextResponse
	err := json.Unmarshal(raw, &nextResponse)
	if err != nil {
		return nil, fmt.Errorf("music: error parsing next response: %w", err)
	}

	searchResults := make([]SearchResult, 0)
	contents := nextResponse.Contents.SingleColumnMusicWatchNextResultsRenderer.TabbedRenderer.WatchNextTabbedResultsRenderer.Tabs[0].TabRenderer.Content.MusicQueueRenderer.Content.PlaylistPanelRenderer.Contents
	for _, content := range contents {
		renderer := content.PlaylistPanelVideoRenderer

		videoID := strings.TrimSpace(renderer.VideoID)
		if videoID == "" {
			videoID = strings.TrimSpace(renderer.NavigationEndpoint.WatchEndpoint.VideoID)
		}
		longBylineRuns := runsFromSubtitleRaw(renderer.LongBylineText)
		shortBylineRuns := runsFromSubtitleRaw(renderer.ShortBylineText)

		result := SearchResult{
			Title:       firstRunText(renderer.Title.Runs),
			VideoID:     videoID,
			CoverArtUrl: parseThumbnailURL(gjson.GetBytes(renderer.Thumbnail, "thumbnails.0.url").String()),
			Artist: firstNonEmpty(
				extractArtist(longBylineRuns),
				extractArtist(shortBylineRuns),
			),
			Album: firstNonEmpty(
				extractAlbum(longBylineRuns),
				extractAlbum(shortBylineRuns),
			),
		}
		appendSearchResult(&searchResults, result)
	}

	if len(searchResults) == 0 {
		return nil, nil
	}

	return searchResults, nil
}

func parseListItemResult(titleRuns, bylineRuns []Run, videoID, coverArtURL string) SearchResult {
	return SearchResult{
		CoverArtUrl: coverArtURL,
		Title:       firstRunText(titleRuns),
		VideoID:     strings.TrimSpace(videoID),
		Artist: firstNonEmpty(
			extractArtist(bylineRuns),
			extractArtist(titleRuns),
		),
		Album: firstNonEmpty(
			extractAlbum(bylineRuns),
			extractAlbum(titleRuns),
		),
	}
}

func flexColumnRuns[T any](columns []T, idx int) []Run {
	if idx < 0 || idx >= len(columns) {
		return nil
	}
	switch c := any(columns[idx]).(type) {
	case struct {
		MusicResponsiveListItemFlexColumnRenderer struct {
			Text struct {
				Runs []Run `json:"runs"`
			} `json:"text"`
			DisplayPriority string `json:"displayPriority"`
		} `json:"musicResponsiveListItemFlexColumnRenderer"`
	}:
		return c.MusicResponsiveListItemFlexColumnRenderer.Text.Runs
	default:
		return nil
	}
}

func firstRunText(runs []Run) string {
	if len(runs) == 0 {
		return ""
	}
	return strings.TrimSpace(runs[0].Text)
}

func runsFromSubtitleRaw(raw json.RawMessage) []Run {
	if len(raw) == 0 {
		return nil
	}

	var subtitle Subtitle
	if err := json.Unmarshal(raw, &subtitle); err != nil {
		return nil
	}
	return subtitle.Runs
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func appendSearchResult(results *[]SearchResult, result SearchResult) {
	if strings.TrimSpace(result.VideoID) == "" || strings.TrimSpace(result.Title) == "" {
		return
	}
	*results = append(*results, result)
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

	if strings.Contains(url, "googleusercontent.com") {
		lastEq := strings.LastIndex(url, "=")
		if lastEq == -1 {
			return url
		}

		// Ensure it's not part of a query parameter
		if strings.Contains(url[lastEq:], "&") {
			return url
		}

		return url[:lastEq] + "=s0-rj"
	}

	return url

}
