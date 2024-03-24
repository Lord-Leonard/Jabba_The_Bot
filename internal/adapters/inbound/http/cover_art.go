package http

import "strings"

func youtubeCoverURL(videoID string) string {
	trimmed := strings.TrimSpace(videoID)
	if trimmed == "" {
		return ""
	}
	// maxresdefault is not guaranteed to exist for every YouTube video.
	// hqdefault is much more reliable and avoids frequent broken thumbnails.
	return "https://i.ytimg.com/vi/" + trimmed + "/hqdefault.jpg"
}
