package spotify

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
)

type SearchResult struct {
	Title, Uploader, URL, Duration, ID string
	Live                               bool
	SourceName                         string
	Extra                              []string
}

func getYoutubeID(track Track) (string, error) {
	var durationMatchTolerance = 15 // seconds
	songDuration := track.Duration
	searchQuery := track.Title + " " + track.Artist + "audio"

	ytSearchRes, err := youtubeSearch(searchQuery, 4)
	if err != nil {
		return "", err
	}
	for _, item := range ytSearchRes {
		allowedDurationRangeStart := songDuration - durationMatchTolerance
		allowedDurationRangeEnd := songDuration + durationMatchTolerance
		resultDuration := convertStringDurationToSeconds(item.Duration)
		if resultDuration <= allowedDurationRangeEnd && resultDuration >= allowedDurationRangeStart {
			return item.ID, nil
		}
	}
	return "", nil
}

func convertStringDurationToSeconds(durationStr string) int {
	splitEntities := strings.Split(durationStr, ":")
	if len(splitEntities) == 1 {
		seconds, _ := strconv.Atoi(splitEntities[0])
		return seconds
	} else if len(splitEntities) == 2 {
		seconds, _ := strconv.Atoi(splitEntities[1])
		minutes, _ := strconv.Atoi(splitEntities[0])
		return (minutes * 60) + seconds
	} else if len(splitEntities) == 3 {
		seconds, _ := strconv.Atoi(splitEntities[2])
		minutes, _ := strconv.Atoi(splitEntities[1])
		hours, _ := strconv.Atoi(splitEntities[0])
		return ((hours * 60) * 60) + (minutes * 60) + seconds
	} else {
		return 0
	}
}

func getContent(data []byte, index int) []byte {
	id := fmt.Sprintf("[%d]", index)
	contents, _, _, _ := jsonparser.Get(data, "contents", "twoColumnSearchResultsRenderer", "primaryContents", "sectionListRenderer", "contents", id, "itemSectionRenderer", "contents")
	return contents
}

func youtubeSearch(searchQuery string, limitResult int) (results []*SearchResult, err error) {
	ytSearchURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(searchQuery)
	req, err := http.NewRequest("GET", ytSearchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("accept-language", "en")
	httpClient := &http.Client{}
	res, err := httpClient.Do(req)

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch YouTube search results")
	}

	buffer, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New("failed to read response body")
	}
	body := string(buffer)
	splitScript := strings.Split(body, `window["ytInitialData"] = `)
	if len(splitScript) != 2 {
		splitScript = strings.Split(body, `var ytInitialData = `)
	}
	if len(splitScript) != 2 {
		return nil, errors.New("failed to parse YouTube search results")
	}
	splitScript = strings.Split(splitScript[1], `window["ytInitialPlayerResponse"] = null;`)
	jsonData := []byte(splitScript[0])
	index := 0
	var contents []byte
	for {
		contents = getContent(jsonData, index)
		_, _, _, err = jsonparser.Get(contents, "[0]", "carouselAdRenderer")

		if err == nil {
			index++
		} else {
			break
		}
	}

	_, err = jsonparser.ArrayEach(contents, func(value []byte, t jsonparser.ValueType, i int, err error) {
		if err != nil {
			return
		}

		if limitResult > 0 && len(results) >= limitResult {
			return
		}

		id, err := jsonparser.GetString(value, "videoRenderer", "videoId")
		if err != nil {
			return
		}

		title, err := jsonparser.GetString(value, "videoRenderer", "title", "runs", "[0]", "text")
		if err != nil {
			return
		}

		uploader, err := jsonparser.GetString(value, "videoRenderer", "ownerText", "runs", "[0]", "text")
		if err != nil {
			return
		}

		live := false
		duration, err := jsonparser.GetString(value, "videoRenderer", "lengthText", "simpleText")

		if err != nil {
			duration = ""
			live = true
		}

		results = append(results, &SearchResult{
			Title:      title,
			Uploader:   uploader,
			Duration:   duration,
			ID:         id,
			URL:        fmt.Sprintf("https://youtube.com/watch?v=%s", id),
			Live:       live,
			SourceName: "youtube",
		})
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}
