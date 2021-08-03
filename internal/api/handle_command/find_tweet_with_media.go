package handle_command

import (
	"context"
	"fmt"

	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
)

const MAX_DEPTH = 2

type ErrWrongMediaType struct {
	err error
}

func (e *ErrWrongMediaType) Error() string {
	return e.err.Error()
}

func (e *ErrWrongMediaType) Unwrap() error {
	return e.err
}

type ErrNoPhotosFound struct {
	err error
}

func (e *ErrNoPhotosFound) Error() string {
	return e.err.Error()
}

func (e *ErrNoPhotosFound) Unwrap() error {
	return e.err
}

func findTweetWithMedia(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet) (*twitter.Tweet, error) {
	return findTweetWithMediaHelper(ctx, client, tweet, false, MAX_DEPTH)
}

func findTweetWithMediaHelper(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet, didRefresh bool, depth int) (*twitter.Tweet, error) {
	var foundTweet *twitter.Tweet
	var err error
	if depth < 0 {
		err = &ErrNoPhotosFound{fmt.Errorf("no photo to search for %s", tweet.Id)}
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
				err = &ErrWrongMediaType{fmt.Errorf("tweet %s contains media %v but no photos", tweet.Id, tweet.Media)}
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

func getParentTweet(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet) (*twitter.Tweet, error) {
	var parentTweet *twitter.Tweet
	var err error
	if tweet.Type == twitter.QuoteTweet && tweet.QuoteTweet != nil {
		parentTweet = tweet.QuoteTweet
	} else if tweet.ParentTweetId != "" {
		parentTweet, err = client.GetTweet(ctx, tweet.ParentTweetId)
	} else {
		err = &ErrNoPhotosFound{fmt.Errorf("no parent photo to search for %s", tweet.Id)}
	}
	return parentTweet, err
}
