package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	responseType mediaResponseType
	reply        string
	err          error
}

func sendReplyForBadMedia(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet, err error) {
	var ErrNoPhotosFoundType *ErrNoPhotosFound
	var ErrWrongMediaTypeType *ErrWrongMediaType
	reply := ""
	if errors.As(err, &ErrNoPhotosFoundType) {
		reply = "I didn't find any photos to interpret, but I appreciate the shoutout!. Try \"@captions_please help\" to learn more"
	} else if errors.As(err, &ErrWrongMediaTypeType) {
		reply = "I only know how to interpret photos right now, sorry!"
	} else {
		reply = "My joints are freezing up! Hey @TheOtherAnil can you please fix me?"
	}
	sendReplies(ctx, client, tweet, []string{reply})
	// Drop the error as we're already handling an error. This is just best-effort
}

func sendReplies(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet, replies []string) error {
	reply := strings.Join(replies, "\n")
	_, err := replyWithMultipleTweets(ctx, client, tweet.Id, reply)

	if err != nil {
		logrus.Info(fmt.Sprintf("Failed to send response %s to tweet %s with error %v", reply, tweet.Id, err))
	}
	return err
}

func addIndexToMessages(messages *[]string) {
	if len(*messages) > 1 {
		for i, message := range *messages {
			// Pesky humans are 1-indexed
			(*messages)[i] = fmt.Sprintf("Image %d: %s", i+1, message)
		}
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

func extractReplies(responses []mediaResponse, handleErr func(response mediaResponse) string) []string {
	replies := make([]string, len(responses))
	for i, response := range responses {
		if response.err == nil {
			replies[i] = response.reply
		} else {
			replies[i] = handleErr(response)
		}
	}
	return replies
}
