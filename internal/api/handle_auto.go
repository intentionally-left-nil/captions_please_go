package api

import (
	"context"
	"fmt"

	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
)

type autoKey int

const theAutoKey autoKey = 0

type autoState struct {
	client twitter.Twitter
}

const longOCRMessageThreshold = 50

func WithAuto(ctx context.Context, client twitter.Twitter) context.Context {
	state := autoState{client: client}
	return setAutoState(ctx, &state)
}

func setAutoState(ctx context.Context, state *autoState) context.Context {
	return context.WithValue(ctx, theAutoKey, state)
}

func getAutoState(ctx context.Context) *autoState {
	return ctx.Value(theAutoKey).(*autoState)
}

func HandleAuto(ctx context.Context, tweet *twitter.Tweet) <-chan ActivityResult {
	state := getAutoState(ctx)
	mediaTweet, err := findTweetWithMedia(ctx, state.client, tweet)
	if err == nil {
		altTextResponses := getAltTextMediaResponse(ctx, tweet, mediaTweet)
		ocrResponses := getOCRMediaResponse(ctx, tweet, mediaTweet)
		describeResponses := getDescribeMediaResponse(ctx, tweet, mediaTweet)

		mergedResponses := make([]mediaResponse, len(mediaTweet.Media))
		for i := 0; i < len(mediaTweet.Media); i++ {
			altTextResponse := altTextResponses[i]
			ocrResponse := ocrResponses[i]
			describeResponse := describeResponses[i]

			if altTextResponse.responseType == doNothingResponse && ocrResponse.responseType == doNothingResponse && describeResponse.responseType == doNothingResponse {
				// The media item isn't a photo. Ignore it
				mergedResponses[i] = altTextResponse
			} else if altTextResponse.err == nil && altTextResponse.responseType == foundAltTextResponse {
				// Prefer the user's caption if it exist
				mergedResponses[i] = altTextResponse
			} else if ocrResponse.err == nil && describeResponse.err == nil && len(ocrResponse.reply) < longOCRMessageThreshold {
				// If there's both OCR and a description, and the OCR text is less than the cutoff
				// then display both
				reply := fmt.Sprintf("%s. It contains the text: %s", describeResponse.reply, ocrResponse.reply)
				mergedResponses[i] = mediaResponse{index: i, responseType: mergedOCRVisionResponse, reply: reply}
			} else if ocrResponse.err == nil {
				mergedResponses[i] = ocrResponse
			} else {
				mergedResponses[i] = describeResponse
			}
		}
		mergedResponses = removeDoNothings(mergedResponses)
		replies := extractReplies(mergedResponses, func(response mediaResponse) string {
			err = response.err
			reply := response.reply
			if reply == "" {
				reply = "I encountered difficulties interpreting the image. Sorry!"
			}
			return reply
		})
		sendErr := sendReplies(ctx, state.client, tweet, replies)
		if err == nil {
			err = sendErr
		}
	} else {
		sendReplyForBadMedia(ctx, state.client, tweet, err)
	}
	out := make(chan ActivityResult, 1)
	out <- ActivityResult{tweet: tweet, err: err, action: "reply with auto response"}
	close(out)
	return out
}
