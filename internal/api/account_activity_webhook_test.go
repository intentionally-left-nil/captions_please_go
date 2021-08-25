package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	twitter_test "github.com/AnilRedshift/captions_please_go/pkg/twitter/test"
	vision_test "github.com/AnilRedshift/captions_please_go/pkg/vision/test"
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
)

func TestWithAccountActivity(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	secrets := &common.Secrets{GooglePrivateKeySecret: vision_test.DummyGoogleCert}
	ctx = common.SetSecrets(ctx, secrets)
	mockTwitter := &twitter_test.MockTwitter{T: t}
	ctx, err := WithAccountActivity(ctx, ActivityConfig{}, mockTwitter)
	state := getActivityState(ctx)
	assert.NotNil(t, state)
	assert.NoError(t, err)
}

func TestAccountActivityWebhook(t *testing.T) {
	// Turn on for debug logging
	// logrus.SetLevel(logrus.DebugLevel)
	botEntity := "\"entities\":{\"user_mentions\":[{\"id_str\":\"123\", \"screen_name\":\"captions_please\", \"name\":\"myName\", \"indices\":[0,16]}]}"
	helpTweet := "{\"id_str\":\"helpTweet\", \"text\": \"@captions_please help\", " + botEntity + "}"
	tests := []struct {
		name               string
		message            string
		maxOutstandingJobs uint
		apiResponse        APIResponse
		timesToDelay       int
		numErrors          int
		expectedActions    []string
	}{
		{
			name:            "Does nothing if the bot is not mentioned",
			message:         "{\"for_user_id\":\"123\", \"tweet_create_events\":[{\"id_str\":\"tweetid\", \"text\": \"hello\"}]}",
			apiResponse:     APIResponse{Status: http.StatusOK},
			expectedActions: []string{"User didnt mention us. Ignoring"},
		},
		{
			name:            "Errors if the JSON payload is not a valid tweet",
			message:         "not a tweet",
			apiResponse:     APIResponse{Status: http.StatusBadRequest},
			expectedActions: []string{"parsing json"},
			numErrors:       1,
		},
		{
			name:            "Errors if the botID is not set",
			message:         "{}",
			apiResponse:     APIResponse{Status: http.StatusBadRequest},
			expectedActions: []string{"parsing json"},
			numErrors:       1,
		},
		{
			name:            "Ignores tweets from blocked users",
			message:         "{\"for_user_id\":\"123\", \"user_has_blocked\": true}",
			apiResponse:     APIResponse{Status: http.StatusOK},
			expectedActions: []string{"ignoring blocked user"},
		},
		{
			name:            "Ignores empty creation data requests",
			message:         "{\"for_user_id\":\"123\", \"tweet_create_events\":[]}",
			apiResponse:     APIResponse{Status: http.StatusOK},
			expectedActions: []string{"no creation events"},
		},
		{
			name:            "Ignores retweets",
			message:         "{\"for_user_id\":\"123\", \"tweet_create_events\":[{\"id_str\":\"retweet\"," + botEntity + ", \"text\": \"@captions_please\", \"indices\": [0,16], \"retweeted_status\":" + helpTweet + "}]}",
			apiResponse:     APIResponse{Status: http.StatusOK},
			expectedActions: []string{"Not responding to a retweet"},
		},
		// TODO replace with a new test
		// {
		// 	name:            "times out if the webhooks are backed up",
		// 	message:         "{\"for_user_id\":\"123\", \"tweet_create_events\":[" + helpTweet + "," + helpTweet + "]}",
		// 	apiResponse:     APIResponse{status: http.StatusOK},
		// 	timesToDelay:    1,
		// 	numErrors:       1,
		// 	expectedActions: []string{"enqueue activity job", "Reply with help message"},
		// },
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer leaktest.Check(t)()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			secrets := &common.Secrets{GooglePrivateKeySecret: vision_test.DummyGoogleCert}
			ctx = common.SetSecrets(ctx, secrets)
			var delayCount uint64
			mockTwitter := &twitter_test.MockTwitter{T: t, TweetReplyMock: func(string, string) (*twitter.Tweet, error) {
				count := atomic.AddUint64(&delayCount, 1)
				if count <= uint64(test.timesToDelay) {
					time.Sleep(time.Millisecond * 200)
				}
				tweet := twitter.Tweet{Id: "234"}
				return &tweet, nil
			}}
			config := ActivityConfig{
				Workers:            1,
				MaxOutstandingJobs: test.maxOutstandingJobs,
				WebhookTimeout:     time.Millisecond * 100,
			}
			ctx, err := WithAccountActivity(ctx, config, mockTwitter)
			assert.NoError(t, err)
			reader := io.NopCloser(strings.NewReader(test.message))
			request := &http.Request{Body: reader}
			resp, out := AccountActivityWebhook(ctx, request)
			assert.Equal(t, test.apiResponse, resp)
			results := []common.ActivityResult{}
			for result := range out {
				results = append(results, result)
			}
			assert.Equal(t, len(test.expectedActions), len(results))
			actions := []string{}
			for _, result := range results {
				actions = append(actions, result.Action)
			}
			for _, expectedAction := range test.expectedActions {
				found := false

				for i, action := range actions {
					if action == expectedAction {
						actions[i] = actions[len(actions)-1]
						actions = actions[:len(actions)-1]
						found = true
						break
					}
				}
				assert.True(t, found, "Looking for action %s in %v", expectedAction, results)
			}
			foundErrors := 0
			for _, result := range results {
				if result.Err != nil {
					foundErrors++
				}
			}
			assert.Equal(t, test.numErrors, foundErrors)
		})
	}
}

func TestGetCommand(t *testing.T) {
	captionsPleaseOffset := len("@captions_please")
	tests := []struct {
		name     string
		tweet    *twitter.Tweet
		mention  *twitter.Mention
		expected string
	}{
		{
			name:     "simple mention has no command",
			tweet:    &twitter.Tweet{VisibleText: "@captions_please", VisibleTextOffset: 0},
			mention:  &twitter.Mention{EndIndex: captionsPleaseOffset},
			expected: "",
		},
		{
			name:     "whitespace is stripped",
			tweet:    &twitter.Tweet{VisibleText: "@captions_please ", VisibleTextOffset: 0},
			mention:  &twitter.Mention{EndIndex: captionsPleaseOffset},
			expected: "",
		},
		{
			name:     "gets a command on one line",
			tweet:    &twitter.Tweet{VisibleText: "@captions_please get alt text", VisibleTextOffset: 0},
			mention:  &twitter.Mention{EndIndex: captionsPleaseOffset},
			expected: "get alt text",
		},
		{
			name:     "ignores text before @captions_please",
			tweet:    &twitter.Tweet{VisibleText: "@other_bot something @captions_please get alt text", VisibleTextOffset: 0},
			mention:  &twitter.Mention{EndIndex: captionsPleaseOffset + len("@other_bot something ")},
			expected: "get alt text",
		},
		{
			name:     "Properly indexes the bot mention with hidden text",
			tweet:    &twitter.Tweet{VisibleText: "@captions_please get alt text", VisibleTextOffset: 33},
			mention:  &twitter.Mention{EndIndex: captionsPleaseOffset + 33},
			expected: "get alt text",
		},
		{
			name:     "Ignores text on new lines",
			tweet:    &twitter.Tweet{VisibleText: "@bot1\n@bot2\n@captions_please get alt text\n@bot4", VisibleTextOffset: 0},
			mention:  &twitter.Mention{EndIndex: captionsPleaseOffset + len("@bot1\n@bot2\n")},
			expected: "get alt text",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, getCommand(test.tweet, test.mention))
		})
	}
}
