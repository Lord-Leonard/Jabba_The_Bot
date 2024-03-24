package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tidwall/gjson"
)

type Song struct {
	Title   string
	Artist  string
	VideoID string
}

func main() {
	// Example: "Never Gonna Give You Up"
	currentVideoID := "dQw4w9WgXcQ"

	fmt.Printf("Generating Radio for ID: %s...\n", currentVideoID)

	vidID, err := ResolveToYouTube("USCA21001264")
	if err != nil {
		panic(err)
	}

	fmt.Printf("YouTube video ID: %s\n", vidID)

	//res, _ := Search("daft punk get lucky")
	//for _, r := range res {
	//	fmt.Printf("[%s] %s - %s (%s)\n", r.Type, r.Title, r.Artist, r.VideoID)
	//}

	//queue, err := GetNextSongs(currentVideoID)
	//if err != nil {
	//	panic(err)
	//}
	//
	//// The first item is usually the song you just requested,
	//// the rest are the "Up Next" suggestions.
	//fmt.Println("\n--- UP NEXT ---")
	//for i, song := range queue {
	//	// Skip the first one if it's the current song
	//	if song.VideoID == currentVideoID && i == 0 {
	//		continue
	//	}
	//	fmt.Printf("[%d] %s - %s (%s)\n", i, song.Title, song.Artist, song.VideoID)
	//}
}

func ResolveToYouTube(isrc string) (string, error) {
	// STRATEGY 1: The Precision Strike (ISRC)
	// Wrap in quotes to force exact match
	query := fmt.Sprintf("\"%s\"", isrc)

	// Use the Search function we wrote earlier
	results, _ := Search(isrc)

	// If we find a "Songs" result, it's 99.9% the correct match
	if len(results) > 0 {
		// Optional: Verify title similarity just in case
		//fmt.Printf("ISRC Match found for %s\n", meta.Title)
		return results[0].VideoID, nil
	}

	// STRATEGY 2: The Fallback (Title + Artist)
	// ISRC failed (YouTube didn't index the tag), so we do it the old fashioned way.
	// fmt.Printf("ISRC failed for %s, falling back to name search...\n", meta.Title)

	// Clean up query: "Artist - Title Audio" helps target the official audio
	//query := fmt.Sprintf("%s - %s Audio", meta.Artist, meta.Title)
	results, err := Search(query)
	if err != nil {
		return "", err
	}

	if len(results) > 0 {
		return results[0].VideoID, nil
	}

	return "", fmt.Errorf("no match found")
}

type SearchResult struct {
	Title   string
	Artist  string
	VideoID string
	Type    string // "Song" or "Video"
}

func Search(query string) ([]SearchResult, error) {
	const apiKey = "AIzaSyD_Owy59_EfAHrX36X301Q-897dV9_ayRo"
	const apiUrl = "https://music.youtube.com/youtubei/v1/search?key=" + apiKey

	payload := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "WEB_REMIX",
				"clientVersion": "1.20240101.01.00",
				"hl":            "en",
			},
		},
		"query": query,
	}

	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	// Headers are crucial for consistent results
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	//var root map[string]interface{}
	//if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
	//	return nil, err
	//}

	body, _ := io.ReadAll(resp.Body)
	jsonStr := string(body) // response body
	// Path string
	base := "contents.tabbedSearchResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents.1.musicCardShelfRenderer"

	videoID := gjson.Get(jsonStr, base+".title.runs.0.navigationEndpoint.watchEndpoint.videoId").String()
	title := gjson.Get(jsonStr, base+".title.runs.0.text").String()
	artist := gjson.Get(jsonStr, base+".subtitle.runs.2.text").String()
	duration := gjson.Get(jsonStr, base+".subtitle.runs.4.text").String()

	fmt.Println("Video ID:", videoID)
	fmt.Println("Title:", title)
	fmt.Println("Artist:", artist)
	fmt.Println("Duration:", duration)

	var results []SearchResult

	results = append(results, SearchResult{
		Title:   title,
		Artist:  artist,
		VideoID: videoID,
	})
	return results, nil
}

func GetNextSongs(videoID string) ([]Song, error) {
	// 1. CONSTANTS
	// This is the public API key used by the YouTube Music Web Client.
	// It rarely changes, but if it does, check the source code of music.youtube.com
	const apiKey = "AIzaSyD_Owy59_EfAHrX36X301Q-897dV9_ayRo" // Generic Browser Key
	const apiUrl = "https://music.youtube.com/youtubei/v1/next?key=" + apiKey

	// Magic prefix to trigger "Radio Mode"
	radioPlaylistID := "RDAMVM" + videoID

	payload := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "WEB_REMIX",
				"clientVersion": "1.20240101.01.00",
				"hl":            "en",
			},
		},
		"videoId":                       videoID,
		"playlistId":                    radioPlaylistID, // <--- ADD THIS
		"enablePersistentPlaylistPanel": true,
		"isAudioOnly":                   true,
	}
	jsonPayload, _ := json.Marshal(payload)

	// 3. MAKE REQUEST
	req, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36")
	// If you want to bypass age restrictions or get personalized recs,
	// you would add "Cookie" header here with your browser cookies.

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API Error %d: %s", resp.StatusCode, string(body))
	}

	//// 4. PARSE JSON (The ugly part)
	//// YouTube returns a deeply nested structure. We use a generic map to traverse it.
	//var result map[string]interface{}
	//if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
	//	return nil, err
	//}

	body, _ := io.ReadAll(resp.Body)
	jsonStr := string(body) // response body
	// Path string
	path := "contents.singleColumnMusicWatchNextResultsRenderer.tabbedRenderer.watchNextTabbedResultsRenderer.tabs.0.tabRenderer.content.musicQueueRenderer.content.playlistPanelRenderer.contents"
	results := gjson.Get(jsonStr, path)

	var songs []Song

	results.ForEach(func(key, value gjson.Result) bool {
		title := value.Get("playlistPanelVideoRenderer.title.runs.0.text").String()
		videoId := value.Get("playlistPanelVideoRenderer.videoId").String()
		songs = append(songs, Song{
			Title:   title,
			VideoID: videoId,
		})
		return true
	})

	return songs, nil
}
