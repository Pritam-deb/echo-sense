package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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
	Token  string    `json:"access_token"`
	Expiry time.Time `json:"expiry_at"`
}

func saveToken(token string, expiry int64) error {
	ct := cachedToken{
		Token:  token,
		Expiry: time.Now().Add(time.Duration(expiry) * time.Second),
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

func getCachedToken() (string, error) {
	data, err := os.ReadFile(cachedTokenFile)
	if err != nil {
		return "", err
	}
	var ct cachedToken
	if err := json.Unmarshal(data, &ct); err != nil {
		return "", err
	}
	if time.Now().After(ct.Expiry) {
		return "", fmt.Errorf("cached token expired")
	}
	return ct.Token, nil
}

func GetAccessToken() (string, error) {
	token, err := getCachedToken()
	if err == nil && token != "" {
		return token, nil
	}
	creds, err := loadCreds()
	if err != nil {
		return "", err
	}
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(creds.ClientID+":"+creds.ClientSecret))
	req.Header.Set("Authorization", authHeader)
	// req.URL.RawQuery = data.Encode()
	//print req to console for debugging
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
	return tokenResp.AccessToken, nil
}
