package handle_command

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	twitter_test "github.com/AnilRedshift/captions_please_go/pkg/twitter/test"
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
)

func TestHandleHelp(t *testing.T) {
	helpMessages := []string{`Tag @captions_please in a tweet to interpret the images.
You can customize the response by adding one of the following commands after tagging me:
alt text: See what description the user gave when creating the tweet
get text: Scan the image for text
describe: Use AI to create a`,
		`description of the image
translate: Automatically convert the result to the language code specified. (e.g. translate into ja-jp)`,
	}
	tests := []struct {
		name       string
		twitterErr error
		expected   []string
	}{
		{
			name:     "Replies with a help message",
			expected: helpMessages,
		},
		{
			name:       "Silently ignores reply failures",
			twitterErr: errors.New("failwhale"),
			expected:   helpMessages[:1],
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer leaktest.Check(t)()

			tweetId := 0
			mockTwitter := &twitter_test.MockTwitter{T: t, TweetReplyMock: func(parentId string, message string) (*twitter.Tweet, error) {
				parentAsInt, err := strconv.Atoi(parentId)
				assert.NoError(t, err)
				assert.Equal(t, tweetId, parentAsInt)
				assert.Equal(t, test.expected[tweetId], message)

				tweetId++
				tweet := twitter.Tweet{Id: fmt.Sprintf("%d", tweetId)}
				return &tweet, test.twitterErr
			}}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx, err := replier.WithReplier(ctx, mockTwitter, false)
			assert.NoError(t, err)

			tweet := &twitter.Tweet{Id: "0"}
			result := Help(ctx, tweet)
			assert.Equal(t, common.ActivityResult{Tweet: tweet, Action: "reply with help"}, result)
			assert.Equal(t, len(test.expected), tweetId)
		})

	}
}
