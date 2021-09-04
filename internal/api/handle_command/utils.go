package handle_command

import (
	"context"
	"errors"
	"fmt"

	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
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
	combinedResponse
)

type mediaResponse struct {
	index        int
	responseType mediaResponseType
	reply        message.Localized
	err          structured_error.StructuredError
}

func combinedError(responses []mediaResponse) structured_error.StructuredError {
	for _, response := range responses {
		if response.err != nil {
			return response.err
		}
	}
	return nil
}

// used for mocking
var _reply = replier.Reply

func getReplyMessageFromResponses(ctx context.Context, responses []mediaResponse) message.Localized {
	responses = removeDoNothings(responses)
	if len(responses) == 0 {
		responses = []mediaResponse{{err: structured_error.Wrap(errors.New("nothing to do when sending replies"), structured_error.NoPhotosFound)}}
	}

	replies := make([]message.Localized, len(responses))
	for i, response := range responses {
		var reply message.Localized
		if response.err != nil {
			reply = message.ErrorMessage(ctx, response.err)
		} else {
			reply = response.reply
		}

		if len(responses) > 1 {
			reply = message.LabelImage(ctx, reply, response.index)
		}
		replies[i] = reply
	}
	return message.CombineMessages(replies, "\n")
}

func replyWithError(ctx context.Context, tweet *twitter.Tweet, err structured_error.StructuredError) {
	message := message.ErrorMessage(ctx, err)
	errResult := _reply(ctx, tweet, message)
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
