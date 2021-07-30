package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	secrets := &Secrets{GooglePrivateKeySecret: vision_test.DummyGoogleCert}
	ctx = withSecrets(ctx, secrets)
	mockTwitter := &twitter_test.MockTwitter{T: t}
	ctx, err := WithAccountActivity(ctx, ActivityConfig{}, mockTwitter)
	state := getActivityState(ctx)
	assert.NotNil(t, state)
	assert.NoError(t, err)
}

func TestAccountActivityWebhook(t *testing.T) {
	// Turn on for debug logging
	// logrus.SetLevel(logrus.DebugLevel)
	helpTweet := "{\"id_str\":\"helpTweet\", \"text\": \"@captions_please help\", \"entities\":{\"user_mentions\":[{\"id_str\":\"123\", \"screen_name\":\"captions_please\", \"name\":\"myName\", \"indices\":[0,16]}]}}"
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
			apiResponse:     APIResponse{status: http.StatusOK},
			expectedActions: []string{"User didnt mention us. Ignoring"},
		},
		{
			name:            "Errors if the JSON payload is not a valid tweet",
			message:         "not a tweet",
			apiResponse:     APIResponse{status: http.StatusBadRequest},
			expectedActions: []string{"parsing json"},
			numErrors:       1,
		},
		{
			name:            "Errors if the botID is not set",
			message:         "{}",
			apiResponse:     APIResponse{status: http.StatusBadRequest},
			expectedActions: []string{"parsing json"},
			numErrors:       1,
		},
		{
			name:            "Ignores tweets from blocked users",
			message:         "{\"for_user_id\":\"123\", \"user_has_blocked\": true}",
			apiResponse:     APIResponse{status: http.StatusOK},
			expectedActions: []string{"ignoring blocked user"},
		},
		{
			name:            "Ignores empty creation data requests",
			message:         "{\"for_user_id\":\"123\", \"tweet_create_events\":[]}",
			apiResponse:     APIResponse{status: http.StatusOK},
			expectedActions: []string{"no creation events"},
		},
		{
			name:            "times out if the webhooks are backed up",
			message:         "{\"for_user_id\":\"123\", \"tweet_create_events\":[" + helpTweet + "," + helpTweet + "]}",
			apiResponse:     APIResponse{status: http.StatusOK},
			timesToDelay:    1,
			numErrors:       1,
			expectedActions: []string{"enqueue activity job", "Reply with help message"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer leaktest.Check(t)()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			secrets := &Secrets{GooglePrivateKeySecret: vision_test.DummyGoogleCert}
			ctx = withSecrets(ctx, secrets)
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
			results := []ActivityResult{}
			for result := range out {
				results = append(results, result)
			}
			assert.Equal(t, len(test.expectedActions), len(results))
			actions := []string{}
			for _, result := range results {
				actions = append(actions, result.action)
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
				if result.err != nil {
					foundErrors++
				}
			}
			assert.Equal(t, test.numErrors, foundErrors)
		})
	}

}
