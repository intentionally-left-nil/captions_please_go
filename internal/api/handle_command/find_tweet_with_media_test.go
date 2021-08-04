package handle_command

import (
	"context"
	"errors"
	"testing"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	twitter_test "github.com/AnilRedshift/captions_please_go/pkg/twitter/test"
	"github.com/stretchr/testify/assert"
)

func TestFindTweetWithMedia(t *testing.T) {
	anError := errors.New("mainstream media, what a silly term")
	mixedMedia := []twitter.Media{{Type: "photo"}, {Type: "video"}}
	onePhoto := []twitter.Media{{Type: "photo"}}
	oneVideo := []twitter.Media{{Type: "video"}}
	oneGif := []twitter.Media{{Type: "animated_gif"}}

	tweetWithMedia := twitter.Tweet{Id: "withMedia", Media: mixedMedia, FallbackMedia: oneVideo}
	otherTweetWithMedia := twitter.Tweet{Id: "withMedia2", Media: mixedMedia, FallbackMedia: oneVideo}
	tweetWithParent := twitter.Tweet{Id: "withParent", ParentTweetId: "tweetWithMedia"}
	tweetWithVideo := twitter.Tweet{Id: "WithVideo", Media: oneVideo, FallbackMedia: oneVideo}
	tweetWithoutMedia := twitter.Tweet{Id: "NoMedia"}
	tweetWithGif := twitter.Tweet{Id: "withGif", Media: oneGif, FallbackMedia: onePhoto}
	quoteImage := twitter.Tweet{Id: "quoteImage", QuoteTweet: &tweetWithMedia, Type: twitter.QuoteTweet}
	quoteTweetWithGrandparentImage := twitter.Tweet{Id: "quoteTweetWithGrandparentImage", Type: twitter.QuoteTweet, QuoteTweet: &tweetWithParent}
	quoteTweetWithFallbackMedia := twitter.Tweet{Id: "quoteTweetWithFallback", FallbackMedia: onePhoto, Type: twitter.QuoteTweet}

	tests := []struct {
		name            string
		tweet           *twitter.Tweet
		parents         []*twitter.Tweet
		refreshedTweets map[string]*twitter.Tweet
		expected        *twitter.Tweet
		err             structured_error.StructuredError
		getTweetErr     error
	}{
		{
			name:     "Returns the current tweet if it contains photos",
			tweet:    &tweetWithMedia,
			expected: &tweetWithMedia,
		},
		{
			name:     "Returns the media of the parent tweet",
			tweet:    &tweetWithoutMedia,
			parents:  []*twitter.Tweet{&tweetWithMedia},
			expected: &tweetWithMedia,
		},
		{
			name:            "Refreshes the initial tweet if it is missing media",
			tweet:           &quoteTweetWithFallbackMedia,
			refreshedTweets: map[string]*twitter.Tweet{"quoteTweetWithFallback": &tweetWithMedia},
			expected:        &tweetWithMedia,
		},
		{
			name:     "Returns the media of the grandparent tweet",
			tweet:    &tweetWithoutMedia,
			parents:  []*twitter.Tweet{&tweetWithoutMedia, &tweetWithMedia},
			expected: &tweetWithMedia,
		},
		{
			name:     "Returns the quote tweet if it contains media",
			tweet:    &quoteImage,
			parents:  []*twitter.Tweet{&otherTweetWithMedia},
			expected: &tweetWithMedia,
		},
		{
			name:    "Errors if the media is too high (great-great-grandparent)",
			tweet:   &tweetWithoutMedia,
			parents: []*twitter.Tweet{&tweetWithoutMedia, &tweetWithoutMedia, &tweetWithoutMedia, &tweetWithMedia},
			err:     structured_error.Wrap(anError, structured_error.NoPhotosFound),
		},
		{
			name:  "Errors out if its a tweet without a parent",
			tweet: &tweetWithoutMedia,
			err:   structured_error.Wrap(anError, structured_error.NoPhotosFound),
		},
		{
			name:  "Errors out if its a tweet with videos but not photos",
			tweet: &tweetWithVideo,
			err:   structured_error.Wrap(anError, structured_error.WrongMediaType),
		},
		{
			name:  "Errors for a tweet with an animated_gif and a fallback photo",
			tweet: &tweetWithGif,
			err:   structured_error.Wrap(anError, structured_error.WrongMediaType),
		},
		{
			name:  "Ignores the quote tweet's parent for the image",
			tweet: &quoteTweetWithGrandparentImage,
			// the quote tweet doesn't need to make a request
			parents: []*twitter.Tweet{&tweetWithMedia},
			err:     structured_error.Wrap(anError, structured_error.NoPhotosFound),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parentIndex := 0
			ctx := context.Background()
			mockTwitter := &twitter_test.MockTwitter{T: t, GetTweetMock: func(parentId string) (*twitter.Tweet, error) {
				var tweet *twitter.Tweet
				if refreshedTweet, ok := test.refreshedTweets[parentId]; ok && parentId != "" {
					tweet = refreshedTweet
				} else {
					assert.Less(t, parentIndex, len(test.parents))
					tweet = test.parents[parentIndex]
					parentIndex++

					if parentIndex < len(test.parents) {
						tweet.ParentTweetId = test.parents[parentIndex].Id
					}
				}
				return tweet, test.getTweetErr
			}}

			if len(test.parents) == 0 {
				test.tweet.ParentTweetId = ""
			} else if test.tweet.Type != twitter.QuoteTweet {
				test.tweet.ParentTweetId = test.parents[0].Id
			}
			tweet, err := findTweetWithMedia(ctx, mockTwitter, test.tweet)
			if test.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, tweet)
			} else {
				assert.Error(t, err)
				assert.Equal(t, test.err.Type(), err.Type())
			}
		})
	}
}
