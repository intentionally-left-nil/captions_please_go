package handle_command

import (
	"context"
	"errors"
	"fmt"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
)

type mediaResponseType int

const (
	doNothingResponse = iota
	foundAltTextResponse
	missingAltTextResponse
	foundOCRResponse
	foundVisionResponse
	mergedOCRVisionResponse
)

type mediaResponse struct {
	index        int
	responseType mediaResponseType
	reply        replier.Localized
	err          structured_error.StructuredError
}

func combineAndSendResponses(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet, getResponses func(context.Context, *twitter.Tweet, *twitter.Tweet) []mediaResponse) common.ActivityResult {
	mediaTweet, err := findTweetWithMedia(ctx, client, tweet)
	tweetToReplyTo := tweet
	if err == nil {
		responses := getResponses(ctx, tweet, mediaTweet)
		combined := combineResponses(ctx, responses)
		result := replier.Reply(ctx, tweet, combined)
		if result.Err == nil {
			return common.ActivityResult{Tweet: tweet, Err: combinedError(responses)}
		}
		err = result.Err
		tweetToReplyTo = result.ParentTweet
	}
	replyWithError(ctx, tweetToReplyTo, err)
	return common.ActivityResult{Tweet: tweet, Err: err}
}

func combinedError(responses []mediaResponse) structured_error.StructuredError {
	for _, response := range responses {
		if response.err != nil {
			return response.err
		}
	}
	return nil
}

func combineResponses(ctx context.Context, responses []mediaResponse) (message replier.Localized) {
	responses = removeDoNothings(responses)
	if len(responses) == 0 {
		responses = []mediaResponse{{err: structured_error.Wrap(errors.New("nothing to do when sending replies"), structured_error.NoPhotosFound)}}
	}

	replies := make([]replier.Localized, len(responses))
	for i, response := range responses {
		var reply replier.Localized
		if response.err != nil {
			reply = replier.ErrorMessage(ctx, response.err)
		} else {
			reply = response.reply
		}

		if len(responses) > 1 {
			reply = replier.LabelImage(ctx, reply, response.index)
		}
		replies[i] = reply
	}
	return replier.CombineMessages(replies, "\n")
}

func replyWithError(ctx context.Context, tweet *twitter.Tweet, err structured_error.StructuredError) {
	message := replier.ErrorMessage(ctx, err)
	errResult := replier.Reply(ctx, tweet, message)
	if errResult.Err != nil {
		logrus.Info(fmt.Sprintf("%s Tried to reply with %v but there was an error %v", tweet.Id, message, errResult.Err))
	}
}

func removeDoNothings(responses []mediaResponse) []mediaResponse {
	filtered := make([]mediaResponse, 0, len(responses))
	for _, response := range responses {
		if response.responseType != doNothingResponse {
			filtered = append(filtered, response)
		}
	}
	return filtered
}
