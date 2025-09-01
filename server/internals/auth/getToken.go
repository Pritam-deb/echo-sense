package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/Pritam-deb/echo-sense/utils"
)

const (
	tokenURL        = "https://accounts.spotify.com/api/token"
	cachedTokenFile = "spotify_token.json"
)

type creds struct {
	ClientID, ClientSecret string
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type cachedToken struct {
	Token  string `json:"access_token"`
	Expiry int64  `json:"expiry"`
}

func saveToken(token string, expiry int64) error {
	ct := cachedToken{
		Token:  token,
		Expiry: expiry,
	}
	data, err := json.Marshal(ct)
	if err != nil {
		return err
	}
	return os.WriteFile(cachedTokenFile, data, 0600)
}

func loadCreds() (*creds, error) {
	clientID := utils.GetEnv("SPOTIFY_CLIENT_ID", "")
	clientSecret := utils.GetEnv("SPOTIFY_CLIENT_SECRET", "")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("SPOTIFY_CLIENT_ID or SPOTIFY_CLIENT_SECRET variables not set in .env file")
	}
	return &creds{ClientID: clientID, ClientSecret: clientSecret}, nil
}

func GetAccessToken() (string, error) {
	creds, err := loadCreds()
	if err != nil {
		return "", err
	}
	data := url.Values{}
	data.Set("client_id", creds.ClientID)
	data.Set("client_secret", creds.ClientSecret)
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = data.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token: %s", resp.Status)
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if err := saveToken(tokenResp.AccessToken, int64(tokenResp.ExpiresIn)); err != nil {
		return "", err
	}
	fmt.Println("Access token obtained and saved.")
	fmt.Println("Token:", tokenResp.AccessToken)
	return tokenResp.AccessToken, nil
}
