package replier

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/twitter-text-go/validate"
	"github.com/sirupsen/logrus"
)

var parseTweet = validate.ParseTweet
var parseTweetSecondPass = validate.ParseTweet

func splitMessage(message string) ([]string, structured_error.StructuredError) {
	if !utf8.ValidString(message) {
		logrus.Debug("Not a valid utf-8 string")
		return nil, structured_error.Wrap(validate.InvalidCharacterError{}, structured_error.CannotSplitMessage)
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
		if start == end {
			// We tried to repeat, but didn't make any progress. Skip the loop to prevent
			// a false error trying to parse an empty text
			continue
		}
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
				logrus.Debug(fmt.Sprintf("splitMessage errored when calling parseTweet for [%d:%d]%s", start, end, message[start:end]))
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
						logrus.Debug(fmt.Sprintf("splitMessage errored when backtracking to find a cutoff point at index %d for %s", secondPassIndex, message[start:end+runeLen]))
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

	if err != nil {
		logrus.Debug(fmt.Sprintf("splitMessage error %v", err))
	}
	return tweets, structured_error.Wrap(err, structured_error.CannotSplitMessage)
}

func appendTweet(tweets []string, tweet string) []string {
	tweet = strings.TrimSpace(tweet)
	if len(tweet) > 0 {
		tweets = append(tweets, tweet)
	}
	return tweets
}
