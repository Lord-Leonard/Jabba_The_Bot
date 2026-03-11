package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	coverCacheDir   = "library/cache/covers"
	coverMaxBytes   = 8 * 1024 * 1024
	coverFetchLimit = 10 * time.Second
)

var (
	errInvalidCoverURL  = errors.New("invalid cover URL")
	errInvalidCoverMIME = errors.New("upstream response is not an image")
)

func (s *Server) handleCoverArt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	rawURL := strings.TrimSpace(r.URL.Query().Get("u"))
	if rawURL == "" {
		http.Error(w, "missing query param: u", http.StatusBadRequest)
		return
	}

	canonicalURL, err := normalizeCoverURL(rawURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cacheKey := coverCacheKey(canonicalURL)
	etag := fmt.Sprintf(`"%s"`, cacheKey)

	if match := strings.TrimSpace(r.Header.Get("If-None-Match")); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	content, contentType, status, err := fetchOrLoadCover(r.Context(), canonicalURL, cacheKey)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400, stale-while-revalidate=604800")
	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func coverProxyURL(rawCoverURL, videoID string) string {
	sourceURL := strings.TrimSpace(rawCoverURL)
	if sourceURL == "" {
		sourceURL = youtubeCoverURL(videoID)
	}
	if sourceURL == "" {
		return ""
	}
	return "/api/cover?u=" + url.QueryEscape(sourceURL)
}

func fetchOrLoadCover(ctx context.Context, canonicalURL, cacheKey string) ([]byte, string, int, error) {
	if cachedBody, cachedType, ok := loadCoverFromCache(cacheKey); ok {
		return cachedBody, cachedType, http.StatusOK, nil
	}

	client := &http.Client{Timeout: coverFetchLimit}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, canonicalURL, nil)
	if err != nil {
		return nil, "", http.StatusBadRequest, errInvalidCoverURL
	}
	req.Header.Set("User-Agent", "JabbaTheBot/cover-proxy")
	req.Header.Set("Accept", "image/*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", http.StatusBadGateway, fmt.Errorf("upstream cover request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, "", http.StatusTooManyRequests, fmt.Errorf("upstream cover request was rate-limited")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", http.StatusBadGateway, fmt.Errorf("upstream cover request returned %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, coverMaxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", http.StatusBadGateway, fmt.Errorf("read upstream cover response")
	}
	if len(body) == 0 {
		return nil, "", http.StatusBadGateway, fmt.Errorf("upstream cover response was empty")
	}
	if len(body) > coverMaxBytes {
		return nil, "", http.StatusBadGateway, fmt.Errorf("upstream cover image exceeds max size")
	}

	contentType := sanitizeContentType(resp.Header.Get("Content-Type"), body)
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return nil, "", http.StatusBadGateway, errInvalidCoverMIME
	}

	_ = storeCoverInCache(cacheKey, body, contentType)
	return body, contentType, http.StatusOK, nil
}

func normalizeCoverURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errInvalidCoverURL
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed == nil {
		return "", errInvalidCoverURL
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return "", fmt.Errorf("%w: scheme must be https", errInvalidCoverURL)
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" || !isAllowedCoverHost(host) {
		return "", fmt.Errorf("%w: host is not allowed", errInvalidCoverURL)
	}

	parsed.Host = host
	parsed.Fragment = ""
	return parsed.String(), nil
}

func isAllowedCoverHost(host string) bool {
	return host == "i.ytimg.com" ||
		strings.HasSuffix(host, ".ytimg.com") ||
		host == "lh3.googleusercontent.com" ||
		strings.HasSuffix(host, ".googleusercontent.com") ||
		host == "yt3.ggpht.com" ||
		strings.HasSuffix(host, ".ggpht.com")
}

func coverCacheKey(canonicalURL string) string {
	sum := sha256.Sum256([]byte(canonicalURL))
	return hex.EncodeToString(sum[:])
}

func coverCachePaths(cacheKey string) (string, string) {
	imagePath := filepath.Join(coverCacheDir, cacheKey+".img")
	typePath := filepath.Join(coverCacheDir, cacheKey+".type")
	return imagePath, typePath
}

func loadCoverFromCache(cacheKey string) ([]byte, string, bool) {
	imagePath, typePath := coverCachePaths(cacheKey)

	body, err := os.ReadFile(imagePath)
	if err != nil || len(body) == 0 {
		return nil, "", false
	}

	contentTypeRaw, err := os.ReadFile(typePath)
	if err != nil {
		return body, sanitizeContentType("", body), true
	}

	return body, sanitizeContentType(string(contentTypeRaw), body), true
}

func storeCoverInCache(cacheKey string, body []byte, contentType string) error {
	if err := os.MkdirAll(coverCacheDir, 0o755); err != nil {
		return err
	}

	imagePath, typePath := coverCachePaths(cacheKey)
	if err := os.WriteFile(imagePath, body, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(typePath, []byte(contentType), 0o644); err != nil {
		return err
	}
	return nil
}

func sanitizeContentType(headerValue string, fallbackBytes []byte) string {
	contentType := strings.TrimSpace(headerValue)
	if contentType != "" {
		if semi := strings.Index(contentType, ";"); semi >= 0 {
			contentType = strings.TrimSpace(contentType[:semi])
		}
	}
	if contentType == "" {
		contentType = http.DetectContentType(fallbackBytes)
	}
	return contentType
}

func youtubeCoverURL(videoID string) string {
	trimmed := strings.TrimSpace(videoID)
	if trimmed == "" {
		return ""
	}
	// maxresdefault is not guaranteed to exist for every YouTube video.
	// hqdefault is much more reliable and avoids frequent broken thumbnails.
	return "https://i.ytimg.com/vi/" + trimmed + "/hqdefault.jpg"
}
