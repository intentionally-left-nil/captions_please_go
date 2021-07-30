package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
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
	messages := []string{}
	var ErrNoPhotosFoundType *ErrNoPhotosFound
	var ErrWrongMediaTypeType *ErrWrongMediaType
	reply := ""
	if err == nil {
		for _, media := range mediaTweet.Media {
			if media.AltText != nil {
				messages = append(messages, *media.AltText)
			} else if media.Type == "photo" {
				messages = append(messages, noAltText(tweet))
			}
		}
		if len(messages) > 1 {
			for i, message := range messages {
				// Pesky humans are 1-indexed
				messages[i] = fmt.Sprintf("Image %d: %s", i+1, message)
			}
		}

		reply = strings.Join(messages, "\n")
	} else if errors.As(err, &ErrNoPhotosFoundType) {
		reply = "I didn't find any photos to interpret, but I appreciate the shoutout!. Try \"@captions_please help\" to learn more"
	} else if errors.As(err, &ErrWrongMediaTypeType) {
		reply = "I only know how to interpret photos right now, sorry!"
	} else {
		reply = "My joints are freezing up! Hey @TheOtherAnil can you please fix me?"
	}

	// Even if there's an error we want to try and send a response
	_, sendErr := replyWithMultipleTweets(ctx, state.client, tweet.Id, reply)

	if sendErr != nil {
		logrus.Info(fmt.Sprintf("Failed to send response %s to tweet %s with error %v", reply, tweet.Id, sendErr))

		if err == nil {
			err = sendErr
		}
	}

	out := make(chan ActivityResult, 1)
	out <- ActivityResult{tweet: tweet, err: err, action: "reply with alt text"}
	close(out)
	return out
}

func getAltTextState(ctx context.Context) *altTextState {
	return ctx.Value(theAltTextKey).(*altTextState)
}

func noAltText(tweet *twitter.Tweet) string {
	return fmt.Sprintf("%s didn't provide any alt text when posting the image", tweet.User.Display)
}
