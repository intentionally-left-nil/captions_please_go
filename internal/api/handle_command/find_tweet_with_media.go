package handle_command

import (
	"context"
	"errors"
	"fmt"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
)

const MAX_DEPTH = 3

func findTweetWithMedia(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet) (*twitter.Tweet, structured_error.StructuredError) {
	return findTweetWithMediaHelper(ctx, client, tweet, false, MAX_DEPTH)
}

func findTweetWithMediaHelper(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet, didRefresh bool, depth int) (*twitter.Tweet, structured_error.StructuredError) {
	var foundTweet *twitter.Tweet
	var err structured_error.StructuredError
	if depth < 0 {
		err = structured_error.Wrap(errors.New("MAX_DEPTH reached"), structured_error.NoPhotosFound)
	}

	if err == nil {
		photos := getPhotos(tweet.Media)
		if len(tweet.Media) == 0 && len(tweet.FallbackMedia) > 0 && !didRefresh {
			// We may have gotten incomplete information from twitter.
			// Re-query the API for the tweet, passing in tweet_mode=extended
			didRefresh = true
			refreshedTweet, err := client.GetTweet(ctx, tweet.Id)
			if err == nil {
				tweet = refreshedTweet
				photos = getPhotos(tweet.Media)
			}
		}

		if err == nil {
			if len(photos) > 0 {
				foundTweet = tweet
			} else if len(tweet.Media) > 0 {
				err = structured_error.Wrap(errors.New("tweet contains media but no photos"), structured_error.WrongMediaType)
			} else {
				var parentTweet *twitter.Tweet
				parentTweet, err = getParentTweet(ctx, client, tweet)
				if err == nil {
					newDepth := depth - 1
					if tweet.Type == twitter.QuoteTweet {
						// We never travel up higher than one level for a quote tweet as those don't get
						// expanded by the TL
						newDepth = 0
					} else {
						didRefresh = true
					}
					foundTweet, err = findTweetWithMediaHelper(ctx, client, parentTweet, didRefresh, newDepth)
				}
			}
		}
	}
	return foundTweet, err
}

func getPhotos(media []twitter.Media) []twitter.Media {
	photos := []twitter.Media{}
	for _, media := range media {
		if media.Type == "photo" {
			photos = append(photos, media)
		}
	}
	return photos
}

func getParentTweet(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet) (*twitter.Tweet, structured_error.StructuredError) {
	var parentTweet *twitter.Tweet
	var err structured_error.StructuredError
	if tweet.Type == twitter.QuoteTweet && tweet.QuoteTweet != nil {
		logrus.Debug(fmt.Sprintf("%s: Directly returning quote tweet %s", tweet.Id, tweet.QuoteTweet.Id))
		parentTweet = tweet.QuoteTweet
	} else if tweet.ParentTweetId != "" {
		logrus.Debug(fmt.Sprintf("%s: Getting parent tweet %s", tweet.Id, tweet.ParentTweetId))
		parentTweet, err = client.GetTweet(ctx, tweet.ParentTweetId)
	} else {
		logrus.Debug(fmt.Sprintf("%s: No parent tweet", tweet.Id))
		err = structured_error.Wrap(errors.New("no parent tweet"), structured_error.NoPhotosFound)
	}
	return parentTweet, err
}
