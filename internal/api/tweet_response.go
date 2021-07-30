package api

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/AnilRedshift/twitter-text-go/validate"
)

var parseTweet = validate.ParseTweet
var parseTweetSecondPass = validate.ParseTweet

func replyWithMultipleTweets(ctx context.Context, client twitter.Twitter, tweetId string, message string) (*twitter.Tweet, error) {
	var tweet *twitter.Tweet
	messages, err := splitMessage(message)
	if err == nil {
		for _, message := range messages {
			tweet, err = client.TweetReply(ctx, tweetId, message)
			if err == nil {
				break
			}
			tweetId = tweet.Id
		}
	}
	return tweet, err
}

func splitMessage(message string) ([]string, error) {
	if !utf8.ValidString(message) {
		return nil, validate.InvalidCharacterError{}
	}
	_, err := parseTweet(message)
	if err == nil {
		return []string{message}, nil
	}

	// To make sure we break the last tweet properly, add a whitespace to the end. This gets trimmed anyways
	message = message + " "
	tweets := make([]string, 0)
	whitespaceIndex := -1
	whitespaceLen := 0
	start, end, runeValue := 0, 0, rune(0)
	for end, runeValue = range message {
	repeat:
		// Don't need to error check RuneLen since we called utf8.ValidString above
		runeLen := utf8.RuneLen(runeValue)

		if unicode.IsSpace(runeValue) {
			_, err = parseTweet(message[start:end])
			if err == nil {
				// There was a whitespace, but so far it's within the bounds of the tweet length
				// Save the index and continue.
				whitespaceIndex = end
				whitespaceLen = runeLen
			} else if _, ok := err.(validate.TooLongError); !ok {
				// There was some kind of parsing error, give up
				goto finally
			} else if whitespaceIndex >= 0 {
				// The current tweet is too long, but there was a previous whitespace we can
				// break into parts
				tweets = appendTweet(tweets, message[start:whitespaceIndex])
				start = whitespaceIndex + whitespaceLen // Eat the whitespace character as the new tweet serves as the whitespace
				whitespaceIndex = -1
				whitespaceLen = 0
				err = nil
				// We promoted one word, but the text could still be too long.
				// Don't advance to the next rune, but repeat the process with the new start index
				goto repeat
			} else {
				// The entire tweet doesn't contain a whitespace
				// Search from the beginning to find a valid end, then continue
				for secondPassIndex, secondPassRune := range message[start : end+runeLen] {
					secondPassIndex = start + secondPassIndex // secondPassIndex starts at 0 due to the slice
					// but to make the logic simpler we want to use global indices
					_, err = parseTweetSecondPass(message[start : secondPassIndex+utf8.RuneLen(secondPassRune)])
					if _, ok := err.(validate.TooLongError); ok {
						// We found the first index where the string is too long
						// Don't include that rune, and add the string to the list
						tweets = appendTweet(tweets, message[start:secondPassIndex])
						start = secondPassIndex
						whitespaceIndex = -1
						whitespaceLen = 0
						err = nil
						// We promoted one word, but the text could still be too long.
						// Don't advance to the next rune, but repeat the process with the new start index
						goto repeat
					} else if err != nil {
						// There was some kind of parsing error, give up
						goto finally
					}
				}
			}
		}
	}

finally:
	if err == nil && start < end {
		tweets = appendTweet(tweets, message[start:])
	}
	return tweets, err
}

func appendTweet(tweets []string, tweet string) []string {
	tweet = strings.TrimSpace(tweet)
	if len(tweet) > 0 {
		tweets = append(tweets, tweet)
	}
	return tweets
}
