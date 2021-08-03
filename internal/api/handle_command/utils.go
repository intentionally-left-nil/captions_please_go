package handle_command

import (
	"context"
	"fmt"
	"strings"

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
	reply        string
	err          error
}

func sendReplyForBadMedia(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet, err error) {
	reply := ""
	// TODO remove once everything is structured
	sErr := structured_error.Wrap(err, structured_error.Unknown)
	switch sErr.Type() {
	case structured_error.NoPhotosFound:
		reply = "I didn't find any photos to interpret, but I appreciate the shoutout!. Try \"@captions_please help\" to learn more"
	case structured_error.WrongMediaType:
		reply = "I only know how to interpret photos right now, sorry!"
	default:
		reply = "My joints are freezing up! Hey @TheOtherAnil can you please fix me?"
	}
	sendReplies(ctx, client, tweet, []string{reply})
	// Drop the error as we're already handling an error. This is just best-effort
}

func sendReplies(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet, replies []string) error {
	reply := strings.Join(replies, "\n")
	result := replier.Reply(ctx, tweet, replier.Message(reply))

	if result.Err != nil {
		logrus.Info(fmt.Sprintf("Failed to send response %s to tweet %s with error %v", reply, tweet.Id, result.Err))
	}
	return result.Err
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

func extractReplies(responses []mediaResponse, handleErr func(response mediaResponse) string) []string {
	replies := make([]string, len(responses))
	for i, response := range responses {
		if response.err == nil {
			replies[i] = response.reply
		} else {
			replies[i] = handleErr(response)
		}

		if len(responses) > 1 {
			// Pesky humans are 1-indexed
			// Make sure to preserve the original media index, as some media could be filtered out
			replies[i] = fmt.Sprintf("Image %d: %s", response.index+1, replies[i])
		}
	}
	return replies
}
