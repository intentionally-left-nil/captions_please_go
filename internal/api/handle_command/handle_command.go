package handle_command

import (
	"context"
	"errors"
	"fmt"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
)

type commandCtxKey int

const theCommandCtxKey commandCtxKey = 0
const longOCRMessageThreshold = 50

type commandState struct {
	client twitter.Twitter
}

var getAltText = getAltTextMediaResponse
var getOcr = getOCRMediaResponse
var getDescription = getDescribeMediaResponse
var findTweet = findTweetWithMedia

func WithHandleCommand(ctx context.Context, client twitter.Twitter) context.Context {
	state := commandState{client: client}
	return context.WithValue(ctx, theCommandCtxKey, &state)
}

func getHandleCommandState(ctx context.Context) *commandState {
	return ctx.Value(theCommandCtxKey).(*commandState)
}

func HandleCommand(ctx context.Context, commandMessage string, tweet *twitter.Tweet) (result common.ActivityResult) {
	didPanic := true
	defer func() {
		if didPanic {
			replyWithError(ctx, tweet, structured_error.Wrap(errors.New("panic at the disco"), structured_error.Unknown))
		}
	}()
	command := parseCommand(commandMessage)
	logrus.Debug(fmt.Sprintf("running command %v", &command))
	ctx = message.WithLanguage(ctx, command.tag)
	result = handleCommand(ctx, command, tweet)
	didPanic = false
	return result
}

func handleCommand(ctx context.Context, command command, tweet *twitter.Tweet) (result common.ActivityResult) {
	tweetToReplyTo := tweet
	if command.help {
		result = Help(ctx, tweet)
	} else if command.unknown {
		result = Unknown(ctx, tweet)
	} else {

		state := getHandleCommandState(ctx)
		var mediaTweet *twitter.Tweet
		mediaTweet, err := findTweet(ctx, state.client, tweet)
		if err == nil {
			responses := getResponses(ctx, command, mediaTweet)
			combinedResponses := make([]mediaResponse, len(mediaTweet.Media))
			for i := range combinedResponses {
				combinedResponses[i] = combineResponsesForSingleImage(ctx, mediaTweet, i, responses[i])
			}

			replyMessage := getReplyMessageFromResponses(ctx, combinedResponses)
			replyResult := _reply(ctx, tweet, replyMessage)
			if replyResult.Err == nil {
				result = common.ActivityResult{Tweet: tweet, Err: combinedError(combinedResponses)}
			} else {
				err = replyResult.Err
				tweetToReplyTo = replyResult.ParentTweet
			}
		}

		if err != nil {
			replyWithError(ctx, tweetToReplyTo, err)
			result = common.ActivityResult{Tweet: tweet, Err: err}
		}
	}
	return result

}

func doNothings(count int) (responses []mediaResponse) {
	responses = make([]mediaResponse, count)
	for i := range responses {
		responses[i] = mediaResponse{index: i, responseType: doNothingResponse}
	}
	return responses
}

func getResponses(ctx context.Context, command command, mediaTweet *twitter.Tweet) (responses [][]mediaResponse) {
	numMedia := len(mediaTweet.Media)
	responses = make([][]mediaResponse, numMedia)
	var altTextResponses, ocrResponses, describeResponses []mediaResponse
	if command.altText || command.auto {
		altTextResponses = getAltText(ctx, command, mediaTweet)
	} else {
		altTextResponses = doNothings(numMedia)
	}

	if command.ocr || command.auto {
		ocrResponses = getOcr(ctx, command, mediaTweet)
	} else {
		ocrResponses = doNothings(numMedia)
	}

	if command.describe || command.auto {
		describeResponses = getDescription(ctx, mediaTweet)
	} else {
		describeResponses = doNothings(numMedia)
	}

	for i := range responses {
		altTextResponse := altTextResponses[i]
		ocrResponse := ocrResponses[i]
		describeResponse := describeResponses[i]
		hasAltText := altTextResponse.err == nil && altTextResponse.responseType == foundAltTextResponse
		hasOCR := ocrResponse.err == nil && ocrResponse.responseType == foundOCRResponse
		hasDescription := describeResponse.err == nil && describeResponse.responseType == foundVisionResponse
		// This logic is closely intertwined with combineResponsesForSingleImage
		// The returning array for each media has implicit rules, such as the ordering of which type goes first
		// and also whether errors are included.
		if altTextResponse.responseType == doNothingResponse && ocrResponse.responseType == doNothingResponse && describeResponse.responseType == doNothingResponse {
			responses[i] = doNothings(1)
		} else if command.auto {
			if hasAltText {
				responses[i] = []mediaResponse{altTextResponse}
			} else if hasOCR && hasDescription && len(ocrResponse.reply) < longOCRMessageThreshold {
				responses[i] = []mediaResponse{describeResponse, ocrResponse}
			} else if hasOCR {
				responses[i] = []mediaResponse{ocrResponse}
			} else if hasDescription {
				responses[i] = []mediaResponse{describeResponse}

			} else if ocrResponse.err != nil {
				// If there's an error, prefer the OCR one for auto
				responses[i] = []mediaResponse{ocrResponse}
			} else {
				responses[i] = []mediaResponse{describeResponse}
			}
		} else if hasAltText || hasOCR || hasDescription {
			// At least one thing succeeded, so only add things which didn't error
			responses[i] = []mediaResponse{altTextResponse}
			if hasDescription {
				responses[i] = append(responses[i], describeResponse)
			}
			if hasOCR {
				responses[i] = append(responses[i], ocrResponse)
			}
		} else if ocrResponse.responseType != doNothingResponse {
			// Prefer the OCR error
			responses[i] = []mediaResponse{altTextResponse, ocrResponse}
		} else {
			// Fallback to the describe error
			responses[i] = []mediaResponse{altTextResponse, describeResponse}
		}
	}
	return responses
}

func combineResponsesForSingleImage(ctx context.Context, mediaTweet *twitter.Tweet, index int, responses []mediaResponse) (response mediaResponse) {
	responses = removeDoNothings(responses)
	if len(responses) == 0 {
		response = mediaResponse{index: index, responseType: doNothingResponse}
		response.responseType = doNothingResponse
	} else if len(responses) == 1 {
		response = responses[0]
	} else {
		response = mediaResponse{index: index, responseType: combinedResponse}
		var altTextReply, descriptionReply, ocrReply message.Localized
		// Handle the alt text, if it exists
		if responses[0].responseType == foundAltTextResponse {
			altTextReply = message.HasAltText(ctx, mediaTweet.User.Display, string(responses[0].reply))
			responses = responses[1:]
			// alt text can't throw an error, otherwise this needs to be handled here
		} else if responses[0].responseType == missingAltTextResponse {
			altTextReply = responses[0].reply
			responses = responses[1:]
		}

		if len(responses) == 1 && responses[0].err != nil {
			response.err = responses[0].err
			if !altTextReply.IsEmpty() {
				response.reply = message.AddBotError(ctx, altTextReply, response.err)
			}
		} else if len(responses) > 0 {
			for _, r := range responses {
				if r.responseType == foundVisionResponse {
					descriptionReply = r.reply
				} else if r.responseType == foundOCRResponse {
					ocrReply = r.reply
				}
			}
		}

		// There must be > 1 non-empty reply to get this far
		// First we check to see if all 3 are non-empty, otherwise we walk through the remaining 2-reply cases
		if !altTextReply.IsEmpty() && !descriptionReply.IsEmpty() && !ocrReply.IsEmpty() {
			response.reply = message.AddDescription(ctx, altTextReply, message.AddOCR(ctx, descriptionReply, ocrReply))
		} else if altTextReply.IsEmpty() {
			response.reply = message.AddOCR(ctx, descriptionReply, ocrReply)
		} else if descriptionReply.IsEmpty() {
			response.reply = message.AddOCR(ctx, altTextReply, ocrReply)
		} else {
			response.reply = message.AddDescription(ctx, altTextReply, descriptionReply)
		}
	}
	return response
}
