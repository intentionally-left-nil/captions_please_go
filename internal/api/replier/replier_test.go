package replier

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	twitter_test "github.com/AnilRedshift/captions_please_go/pkg/twitter/test"
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
)

func TestWithReplier(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockTwitter := twitter_test.MockTwitter{T: t}
	ctx, err := WithReplier(ctx, &mockTwitter, false)
	assert.NoError(t, err)
	state := getReplierState(ctx)
	assert.NotNil(t, state)
}

func TestReply(t *testing.T) {
	oneHundred := strings.Repeat("a", 100)
	twoHundred := strings.Repeat("b", 200)
	longMessage := twoHundred + " " + oneHundred
	anError := errors.New("oh no")
	twitterError := structured_error.Wrap(anError, structured_error.TwitterError)
	tooLongError := structured_error.Wrap(anError, structured_error.TweetTooLong)
	missingTweetError := structured_error.Wrap(anError, structured_error.CaseOfTheMissingTweet)
	invalidMessage := "\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98"
	tests := []struct {
		name              string
		message           string
		expected          []string
		replyErrs         []structured_error.StructuredError
		shouldCancelEarly bool
		result            ReplyResult
	}{
		{
			name:     "Replies with a message that fits in one tweet",
			message:  "hello",
			expected: []string{"hello"},
			result:   ReplyResult{ParentTweet: &twitter.Tweet{Id: "1"}},
		},
		{
			name:     "Replies with two tweets for a long message",
			message:  longMessage,
			expected: []string{twoHundred, oneHundred},
			result:   ReplyResult{ParentTweet: &twitter.Tweet{Id: "2"}},
		},
		{
			name:    "Errors if parsing the message fails",
			message: invalidMessage,
			result: ReplyResult{
				Err:         structured_error.Wrap(anError, structured_error.CannotSplitMessage),
				ParentTweet: &twitter.Tweet{Id: "0"},
			},
		},
		{
			name:      "Errors if sending the first tweet fails",
			message:   longMessage,
			expected:  []string{twoHundred},
			replyErrs: []structured_error.StructuredError{twitterError},
			result: ReplyResult{
				Err:         twitterError,
				ParentTweet: &twitter.Tweet{Id: "0"},
				Remaining:   []string{twoHundred, oneHundred}},
		},
		{
			name:      "Errors if sending the last tweet fails",
			message:   longMessage,
			expected:  []string{twoHundred, oneHundred},
			replyErrs: []structured_error.StructuredError{nil, twitterError},
			result: ReplyResult{
				Err:         twitterError,
				ParentTweet: &twitter.Tweet{Id: "1"},
				Remaining:   []string{oneHundred}},
		},
		{
			name:      "Splits the tweet into two if twitter says its too long",
			message:   "hello",
			expected:  []string{"hello", "he", "llo"},
			replyErrs: []structured_error.StructuredError{tooLongError, nil, nil},
			result:    ReplyResult{ParentTweet: &twitter.Tweet{Id: "3"}},
		},
		{
			name:      "Retries the tweet with CaseOfTheMissingTweet response",
			message:   "hello",
			expected:  []string{"hello", "hello"},
			replyErrs: []structured_error.StructuredError{missingTweetError, nil},
			result:    ReplyResult{ParentTweet: &twitter.Tweet{Id: "2"}},
		},
		{
			name:      "Times out trying to resend a CaseOfTheMissingTweet response",
			message:   "hello",
			expected:  []string{"hello"},
			replyErrs: []structured_error.StructuredError{missingTweetError},
			result: ReplyResult{
				Err:         missingTweetError,
				ParentTweet: &twitter.Tweet{Id: "0"},
				Remaining:   []string{"hello"},
			},
			shouldCancelEarly: true,
		},
		{
			name:      "Gives up after two CaseOfTheMissingTweets",
			message:   "hello",
			expected:  []string{"hello", "hello"},
			replyErrs: []structured_error.StructuredError{missingTweetError, missingTweetError},
			result: ReplyResult{
				Err:         missingTweetError,
				ParentTweet: &twitter.Tweet{Id: "0"},
				Remaining:   []string{"hello"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			tweetId := 0
			mockTwitter := &twitter_test.MockTwitter{T: t, TweetReplyMock: func(parentId string, message string) (*twitter.Tweet, error) {
				parentAsInt, err := strconv.Atoi(parentId)
				assert.NoError(t, err)
				if test.replyErrs != nil &&
					tweetId > 0 &&
					test.replyErrs[tweetId-1] != nil &&
					(test.replyErrs[tweetId-1].Type() == structured_error.TweetTooLong || test.replyErrs[tweetId-1].Type() == structured_error.CaseOfTheMissingTweet) {
					assert.Equal(t, tweetId-1, parentAsInt)
				} else {
					assert.Equal(t, tweetId, parentAsInt)

				}
				assert.Equal(t, test.expected[tweetId], message)

				if tweetId < len(test.replyErrs) {
					err = test.replyErrs[tweetId]
				} else {
					err = nil
				}

				tweetId++
				tweet := twitter.Tweet{Id: fmt.Sprintf("%d", tweetId)}
				return &tweet, err
			}}
			defer leaktest.Check(t)()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			originalAfter := after
			defer func() {
				after = originalAfter
			}()

			after = func(d time.Duration) <-chan time.Time {
				return time.After(time.Millisecond * 100)
			}

			var earlyTimer *time.Timer

			if test.shouldCancelEarly {
				earlyTimer = time.AfterFunc(time.Millisecond*50, cancel)
			}

			ctx, err := WithReplier(ctx, mockTwitter, false)
			assert.NoError(t, err)
			tweet := &twitter.Tweet{Id: "0"}
			result := Reply(ctx, tweet, message.Unlocalized(test.message))

			if earlyTimer != nil {
				earlyTimer.Stop()
			}
			if test.result.Err == nil {
				assert.NoError(t, result.Err)
			} else {
				assert.Error(t, result.Err)
				assert.Equal(t, test.result.Err.Type(), result.Err.Type())
			}
			assert.Equal(t, test.result.Remaining, result.Remaining)
			assert.Equal(t, test.result.ParentTweet.Id, result.ParentTweet.Id)
			assert.Equal(t, len(test.expected), tweetId)
		})
	}
}
