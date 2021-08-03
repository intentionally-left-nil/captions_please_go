package replier

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

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
	ctx, err := WithReplier(ctx, &mockTwitter)
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
	invalidMessage := "\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98"
	tests := []struct {
		name      string
		message   string
		expected  []string
		replyErrs []error
		result    ReplyResult
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
			replyErrs: []error{twitterError},
			result: ReplyResult{
				Err:         twitterError,
				ParentTweet: &twitter.Tweet{Id: "0"},
				Remaining:   []string{twoHundred, oneHundred}},
		},
		{
			name:      "Errors if sending the last tweet fails",
			message:   longMessage,
			expected:  []string{twoHundred, oneHundred},
			replyErrs: []error{nil, twitterError},
			result: ReplyResult{
				Err:         twitterError,
				ParentTweet: &twitter.Tweet{Id: "1"},
				Remaining:   []string{oneHundred}},
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
			ctx, err := WithReplier(ctx, mockTwitter)
			assert.NoError(t, err)
			tweet := &twitter.Tweet{Id: "0"}
			result := Reply(ctx, tweet, Unlocalized(test.message))
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
