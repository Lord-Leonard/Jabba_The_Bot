package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		panic("error loading .env file")
	}

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		panic("CLIENT_ID or CLIENT_SECRET not set")
	}
	authURL := "https://accounts.spotify.com/api/token"

	// Encode client_id:client_secret as Base64
	credentials := clientID + ":" + clientSecret
	encodedCreds := base64.StdEncoding.EncodeToString([]byte(credentials))

	// Form data
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	// Create request
	req, err := http.NewRequest(
		"POST",
		authURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", "Basic "+encodedCreds)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		panic(fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, body))
	}

	// Parse response
	var result struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		panic(err)
	}

	token := result.AccessToken
	fmt.Println("Access token:", token)
}
