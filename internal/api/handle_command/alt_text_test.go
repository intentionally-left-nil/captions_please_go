package handle_command

import (
	"context"
	"errors"
	"testing"

	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	twitter_test "github.com/AnilRedshift/captions_please_go/pkg/twitter/test"
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
)

func TestWithAltText(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockTwitter := &twitter_test.MockTwitter{T: t}
	ctx = WithAltText(ctx, mockTwitter)
	state := getAltTextState(ctx)
	assert.NotNil(t, state)
}

func TestHandleAltText(t *testing.T) {
	user := twitter.User{Display: "Ada Bear", Id: "999", Username: "@ada_bear"}
	altText := "hello alt text"
	oneVideo := []twitter.Media{{Type: "video"}}
	onePhotoWithAltText := []twitter.Media{{Type: "photo", AltText: &altText}}
	onePhotoWithoutAltText := []twitter.Media{{Type: "photo"}}
	mixedMedia := []twitter.Media{{Type: "photo"}, {Type: "photo", AltText: &altText}, {Type: "video"}}

	tweetWithoutAltText := twitter.Tweet{Id: "withoutMedia", Media: onePhotoWithoutAltText, User: user}
	tweetWithAltText := twitter.Tweet{Id: "withAltText", Media: onePhotoWithAltText, User: user}
	tweetWithMixedMedia := twitter.Tweet{Id: "withAltText", Media: mixedMedia, User: user}
	tweetWithoutMedia := twitter.Tweet{Id: "NoMedia"}
	tweetWithVideo := twitter.Tweet{Id: "WithVideo", Media: oneVideo, FallbackMedia: oneVideo}
	tweetWithParent := twitter.Tweet{Id: "withParent", ParentTweetId: "withMedia"}
	tests := []struct {
		name        string
		tweet       *twitter.Tweet
		replyErr    error
		getTweetErr error
		messages    []string
		hasErr      bool
	}{
		{
			name:     "Responds with the provided alt_text of a single image",
			tweet:    &tweetWithAltText,
			messages: []string{altText},
			hasErr:   false,
		},
		{
			name:     "Responds with no alt text when missing",
			tweet:    &tweetWithoutAltText,
			messages: []string{"Ada Bear didn't provide any alt text when posting the image"},
			hasErr:   false,
		},
		{
			name:     "Responds to a tweet with multiple images",
			tweet:    &tweetWithMixedMedia,
			messages: []string{"Image 1: Ada Bear didn't provide any alt text when posting the image\nImage 2: hello alt text"},
			hasErr:   false,
		},
		{
			name:     "Responds with help message if there's no media",
			tweet:    &tweetWithoutMedia,
			messages: []string{"I didn't find any photos to interpret, but I appreciate the shoutout!. Try \"@captions_please help\" to learn more"},
			hasErr:   true,
		},
		{
			name:     "Responds with appropriate error if its a video",
			tweet:    &tweetWithVideo,
			messages: []string{"I only know how to interpret photos right now, sorry!"},
			hasErr:   true,
		},
		{
			name:        "Responds with a generic error if twitter errors",
			tweet:       &tweetWithParent,
			getTweetErr: errors.New("failwhale"),
			messages:    []string{"My joints are freezing up! Hey @TheOtherAnil can you please fix me?"},
			hasErr:      true,
		},
		{
			name:     "Errors when sending the reply",
			tweet:    &tweetWithAltText,
			messages: []string{altText},
			replyErr: errors.New("failwhale"),
			hasErr:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer leaktest.Check(t)()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			messageIndex := 0
			mockTwitter := &twitter_test.MockTwitter{T: t, TweetReplyMock: func(tweetID string, message string) (*twitter.Tweet, error) {
				assert.Equal(t, test.messages[messageIndex], message)
				messageIndex++
				return &twitter.Tweet{}, test.replyErr
			}, GetTweetMock: func(tweetID string) (*twitter.Tweet, error) {
				return &twitter.Tweet{}, test.getTweetErr
			}}

			ctx = WithAltText(ctx, mockTwitter)
			ctx, err := replier.WithReplier(ctx, mockTwitter)
			assert.NoError(t, err)
			result := <-HandleAltText(ctx, test.tweet)

			assert.Equal(t, len(test.messages), messageIndex)
			if test.hasErr {
				assert.Error(t, result.Err)
			} else {
				assert.NoError(t, result.Err)
				assert.Equal(t, result.Action, "reply with alt text")
			}
		})
	}
}
