package api

import (
	"context"
	"fmt"

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

func HandleAltText(ctx context.Context, tweet *twitter.Tweet) <-chan ActivityResult {
	state := getAltTextState(ctx)
	mediaTweet, err := findTweetWithMedia(ctx, state.client, tweet)
	if err == nil {
		responses := getAltTextMediaResponse(ctx, tweet, mediaTweet)
		responses = removeDoNothings(responses)
		replies := extractReplies(responses, nil) // alt text doesn't produce errorss
		addIndexToMessages(&replies)
		err = sendReplies(ctx, state.client, tweet, replies)
	} else {
		sendReplyForBadMedia(ctx, state.client, tweet, err)
	}
	out := make(chan ActivityResult, 1)
	out <- ActivityResult{tweet: tweet, err: err, action: "reply with alt text"}
	close(out)
	return out
}

func getAltTextMediaResponse(ctx context.Context, tweet *twitter.Tweet, mediaTweet *twitter.Tweet) []mediaResponse {
	responses := make([]mediaResponse, len(mediaTweet.Media))
	for i, media := range mediaTweet.Media {
		var response mediaResponse
		if media.AltText != nil {
			response = mediaResponse{responseType: foundAltTextResponse, reply: *media.AltText}
		} else if media.Type == "photo" {
			reply := fmt.Sprintf("%s didn't provide any alt text when posting the image", tweet.User.Display)
			response = mediaResponse{responseType: missingAltTextResponse, reply: reply}
		} else {
			response = mediaResponse{responseType: doNothingResponse}
		}
		responses[i] = response
	}
	return responses
}

func getAltTextState(ctx context.Context) *altTextState {
	return ctx.Value(theAltTextKey).(*altTextState)
}
