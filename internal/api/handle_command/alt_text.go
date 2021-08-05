package handle_command

import (
	"context"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
)

type altTextKey int

const theAltTextKey altTextKey = 0

type altTextState struct {
	client twitter.Twitter
}

func WithAltText(ctx context.Context, client twitter.Twitter) context.Context {
	state := &altTextState{
		client: client,
	}
	return context.WithValue(ctx, theAltTextKey, state)
}

func HandleAltText(ctx context.Context, tweet *twitter.Tweet) common.ActivityResult {
	state := getAltTextState(ctx)
	response := combineAndSendResponses(ctx, state.client, tweet, getAltTextMediaResponse)
	response.Action = "reply with alt text"
	return response
}

func getAltTextMediaResponse(ctx context.Context, tweet *twitter.Tweet, mediaTweet *twitter.Tweet) []mediaResponse {
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

func getAltTextState(ctx context.Context) *altTextState {
	return ctx.Value(theAltTextKey).(*altTextState)
}
