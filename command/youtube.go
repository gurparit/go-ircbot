package command

import (
	"fmt"
	"net/url"

	"net/http"

	"github.com/gurparit/gobot/env"
	"github.com/gurparit/gobot/httpc"
)

// YoutubeURL base URL for Youtube Search
const YoutubeURL = "https://www.googleapis.com/youtube/v3/search?part=snippet&key=%s&maxResults=1&type=video&q=%s"

// YoutubeVideoURL base URL for Youtube Videos
const YoutubeVideoURL = "%s - http://www.youtube.com/watch?v=%s"

// Youtube the Youtube class
type Youtube struct{}

// YoutubeResult : sample response
type YoutubeResult struct {
	Items []struct {
		Snippet struct {
			Title string `json:"title"`
		} `json:"snippet"`
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
	} `json:"items"`
}

// Execute Youtube implementation
func (Youtube) Execute(r Response, query string) {
	targetURL := fmt.Sprintf(
		YoutubeURL,
		env.OS.GoogleKey,
		url.QueryEscape(query),
	)

	request := httpc.HTTP{
		TargetURL: targetURL,
		Method:    http.MethodGet,
	}

	var result YoutubeResult
	if err := request.JSON(&result); err != nil {
		r(fmt.Sprintf("[youtube] %s", err.Error()))
		return
	}

	resultCount := len(result.Items)

	if resultCount > 0 {
		value := result.Items[0]
		message := fmt.Sprintf(YoutubeVideoURL, value.Snippet.Title, value.ID.VideoID)

		r(message)
	} else {
		r("[youtube] no results found")
	}
}
