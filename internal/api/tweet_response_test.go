package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	twitter_test "github.com/AnilRedshift/captions_please_go/pkg/twitter/test"
	"github.com/AnilRedshift/twitter-text-go/validate"
	"github.com/stretchr/testify/assert"
)

func TestReplyWithMultipleTweets(t *testing.T) {
	oneHundred := strings.Repeat("a", 100)
	twoHundred := strings.Repeat("b", 200)
	tests := []struct {
		name        string
		message     string
		replyErr    error
		shouldError bool
		expected    []string
	}{
		{
			name:     "Replies with a message that fits in one tweet",
			message:  "hello",
			expected: []string{"hello"},
		},
		{
			name:     "Replies with two tweets for a long message",
			message:  twoHundred + " " + oneHundred,
			expected: []string{twoHundred, oneHundred},
		},
		{
			name:        "Errors if parsing the message fails",
			message:     "\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98",
			shouldError: true,
		},
		{
			name:        "Errors if sending the tweet fails",
			message:     "hello",
			replyErr:    errors.New("no sending tweets allowed"),
			expected:    []string{"hello"},
			shouldError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tweetId := 0
			mockTwitter := &twitter_test.MockTwitter{T: t, TweetReplyMock: func(parentId string, message string) (*twitter.Tweet, error) {
				parentAsInt, err := strconv.Atoi(parentId)
				assert.NoError(t, err)
				assert.Equal(t, tweetId, parentAsInt)
				assert.Equal(t, test.expected[tweetId], message)

				tweetId++
				tweet := twitter.Tweet{Id: fmt.Sprintf("%d", tweetId)}
				return &tweet, test.replyErr
			}}

			tweet, err := replyWithMultipleTweets(context.Background(), mockTwitter, "0", test.message)
			assert.Equal(t, len(test.expected), tweetId)
			if test.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("%d", tweetId), tweet.Id)
			}
		})
	}
}

func TestSplitMessage(t *testing.T) {
	fiveCharacterValidate := func(s string) (validate.Tweet, error) {
		tweet := validate.Tweet{}
		if len(s) > 5 {
			return tweet, validate.TooLongError(len(s))
		}
		return tweet, nil
	}

	tenCharacters := "0123456789"
	longMessage := strings.Repeat(tenCharacters, 27) + "ab cde\u3000fg\u3000" + tenCharacters
	longMessageResult := make([]string, 2)
	longMessageResult[0] = strings.Repeat(tenCharacters, 27) + "ab cde\u3000fg"
	longMessageResult[1] = tenCharacters

	myError := errors.New("oops I parsed it again")
	tests := []struct {
		name                 string
		message              string
		parseTweet           func(string) (validate.Tweet, error)
		parseTweetSecondPass func(string) (validate.Tweet, error)
		tweets               []string
		err                  error
	}{
		{
			name:       "Simple case where the message is one valid tweet",
			message:    "hey there",
			parseTweet: func(string) (validate.Tweet, error) { return validate.Tweet{}, nil },
			tweets:     []string{"hey there"},
		},
		{
			name:       "Message breaks into two tweets along whitespace",
			message:    "012 456",
			parseTweet: fiveCharacterValidate,
			tweets:     []string{"012", "456"},
		},
		{
			name:       "Whitespace occurs just before the character limit",
			message:    "0123 56",
			parseTweet: fiveCharacterValidate,
			tweets:     []string{"0123", "56"},
		},
		{
			name:                 "Whitespace occurs on the character limit",
			message:              "01234 67",
			parseTweet:           fiveCharacterValidate,
			parseTweetSecondPass: fiveCharacterValidate,
			tweets:               []string{"01234", "67"},
		},
		{
			name:                 "Whitespace occurs just after the character limit",
			message:              "012345 78",
			parseTweet:           fiveCharacterValidate,
			parseTweetSecondPass: fiveCharacterValidate,
			tweets:               []string{"01234", "5 78"},
		},
		{
			name:       "Messages containing all whitespace are dropped",
			message:    "012  5  8 ",
			parseTweet: fiveCharacterValidate,
			tweets:     []string{"012", "5  8"},
		},
		{
			name:                 "Too long message in the middle of shorter messages",
			message:              "012 456789 123 45",
			parseTweet:           fiveCharacterValidate,
			parseTweetSecondPass: fiveCharacterValidate,
			tweets:               []string{"012", "45678", "9 123", "45"},
		},
		{
			name:                 "Too long message needs to be broken up multiple times",
			message:              "012345678901 34567890 1 2",
			parseTweet:           fiveCharacterValidate,
			parseTweetSecondPass: fiveCharacterValidate,
			tweets:               []string{"01234", "56789", "01", "34567", "890 1", "2"},
		},
		{
			name:                 "Message with multi-byte space character is split properly",
			message:              "0123\u300078\u30002345678901\u3000\u30008\u30009\u30003",
			parseTweet:           fiveCharacterValidate,
			parseTweetSecondPass: fiveCharacterValidate,
			tweets:               []string{"0123", "78", "23456", "78901", "8", "9\u30003"},
		},
		{
			name:                 "Full example with the real API",
			message:              longMessage,
			parseTweet:           parseTweet,
			parseTweetSecondPass: parseTweetSecondPass,
			tweets:               longMessageResult,
		},
		{
			name:    "Message contains an invalid character",
			message: "\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98",
			err:     validate.InvalidCharacterError{},
		},
		{
			name:       "Propagates an error message from validateTweet",
			message:    "hey there",
			parseTweet: func(string) (validate.Tweet, error) { return validate.Tweet{}, myError },
			err:        myError,
		},
		{
			name:                 "Propagates an error message from validateSecondPass",
			message:              "1234567",
			parseTweet:           fiveCharacterValidate,
			parseTweetSecondPass: func(string) (validate.Tweet, error) { return validate.Tweet{}, myError },
			err:                  myError,
		},
		{
			name:                 "Returns EmptyError{} if the string is empty",
			message:              "",
			parseTweet:           parseTweet,
			parseTweetSecondPass: parseTweetSecondPass,
			err:                  validate.EmptyError{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalParseTweet := parseTweet
			originalSecondPass := parseTweetSecondPass
			defer func() {
				parseTweet = originalParseTweet
				parseTweetSecondPass = originalSecondPass
			}()
			parseTweet = test.parseTweet
			parseTweetSecondPass = test.parseTweetSecondPass
			tweets, err := splitMessage(test.message)
			assert.Equal(t, test.err, err)
			if test.err == nil {
				assert.Equal(t, test.tweets, tweets)
			}
		})
	}
}
