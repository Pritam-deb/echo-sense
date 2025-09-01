package spotify

import (
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/Pritam-deb/echo-sense/internals/auth"
)

type Track struct {
	Title, Artist, Album string
	Artists              []string
	Year                 int
	Duration             int // in seconds
}

func hitSpotifyEndpoints(endpoint string) (int, string, error) {
	fmt.Println("Hitting endpoint:", endpoint)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return 0, "", err
	}
	bearerToken, err := auth.GetAccessToken()
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}
	fmt.Println(string(body))
	return resp.StatusCode, resp.Status, nil
}

func GetTrackInfo(url string) (*Track, error) {
	// example url: https://open.spotify.com/track/2VOnMNQWQ44OqHWwvXn5z6\?si\=7f6007e3a57a4706
	baseUrl := "https://api.spotify.com/v1/tracks/"
	pattern := `^(?:https?:\/\/)?open\.spotify\.com\/track\/([A-Za-z0-9]{22})(?:\?.*)?$`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(url)
	var trackID string
	if len(matches) > 1 {
		fmt.Println("Track ID:", matches[1])
		trackID = matches[1]
	} else {
		fmt.Println("No match found")
	}
	endpoint := baseUrl + trackID
	statusCode, _, err := hitSpotifyEndpoints(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error getting track info: %w", err)
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("error from spotify: %d", statusCode)
	}
	return &Track{}, nil
}
