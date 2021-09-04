package handle_command

import (
	"context"

	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
)

func getAltTextMediaResponse(ctx context.Context, mediaTweet *twitter.Tweet) []mediaResponse {
	responses := make([]mediaResponse, len(mediaTweet.Media))
	for i, media := range mediaTweet.Media {
		var response mediaResponse
		if media.AltText != nil {
			response = mediaResponse{index: i, responseType: foundAltTextResponse, reply: message.Unlocalized(*media.AltText)}
		} else if media.Type == "photo" {
			reply := message.NoAltText(ctx, mediaTweet.User.Display)
			response = mediaResponse{index: i, responseType: missingAltTextResponse, reply: reply}
		} else {
			response = mediaResponse{index: i, responseType: doNothingResponse}
		}
		responses[i] = response
	}
	return responses
}
