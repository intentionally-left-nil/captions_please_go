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
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
)

// chill, this isn't used anywhere
const dummyGoogleCert = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAzIVTdjTgQXBdUfnxVavmr8DPJOZiCRMzrSYYJOaYa/qmShwz
FzlNRVRmCsbBv9+ijphTekif5mimxAnxq1is9qOcV/r1TVLoa7aK/yS2814t16re
l5t21JdIYDeMFoIgiH4ynF4C/Z5pRqX41QRofB3qN63ls/+1lRgifVqUdn6A7iIv
p3Y77ZfUPCTcyDWds2F+zLX7DmSpQhlHvIdrZhBZED52PsV2NaCEOQU46e0IaRV6
6/tZLiJ0AMA/fqVaHaXs2LVzq8zdZbLuxGwfe1Bh6FYYPFRwXx5qTPxEv5WFtYU9
xm0Mk9Lnn/AGfOdPkUMJ2V9wXUir9r7vN31OowIDAQABAoIBAQCUOtF56+rZIuJQ
BtIWIKfqm9jGSr+lCii7BtAa9pJkOF8LeZLB80MAy6HFj7ZfJWvA48Ak8bwKl7C+
huKEKJn7jCtFTNs7NqrDXqMxNt/uVUTueaYoxYGDpT3MlpXOvnNr2eM+l5idTpHI
pYRKh45e3qOhxUSlh+CIddyRc/QESHFl9lQ9fZdOKUlHUhDd1iq8iwlC4gTUZH7Q
oOC9k2lfaBL2Eitk2dvQ4c+AJRVyWdNS//ZGHzN6D0p0hx+neqAVAon1aEVD//yS
o9+vnZENQ3faQSf7Fy7mDTXT9TPVQAbItLsVpHfl470yqTJicWMAjZIlnXyaojJO
9EkSYruBAoGBAPGrkECkKNKYOmkPUOmMWrAVRuKCuOEfaqK6QBpZX7nfsRUOYqbe
JL724FlDHJJz5YLMbHl/66O8w3OgdNz+eVpkBNpJlhkQjPF2lApR/rohmdvoOe5j
gs/oCTSdrdEGk1prcCmZ+IwD8GM56JyUYDyS3ZK4NJ1kJppc056yS+bjAoGBANil
2mVC7AcyH+ZH4b23OmFNHwlSIB0VjbcaiHM3vxp4zzzcIX1SJQVsJ5CiqHfEypVn
hc9hHNAhdJ8o5a1IiJ2vYrh0zGpTj7o/3XSkizJcR/Bzn/80BUO26UC86NwFAJ3g
+4ypR8QFhnqOFQ3z/3bieaamByj4Afq+QgsrOcVBAoGBAJmjOFHgCxPXM0sXMZlI
YV8QJ8BY2rBECMbrIVWe+/xu+WUpgA4Vq8a7rGUTBVcV1xMQYuXbLTMrDha0K5dT
MFMGww8DOSk2HGRlvjfRaN9r/SSQvkOPf9os6a1JkPcR9xvEscnA2QIqfuiWKAtj
SMs5kyNzd/+Xa/M2kFKThy2BAoGADF3jSpZ4XKzKz11ZEHhOF9HMLL8IYECjt0kH
cvRCr2MoCURTkRDIVjfnRkVSsouEOOUQ6VaUy3itbIxsF+klC0NAsmDQbl1Yvfv5
Szg9TeGgpaQkBPBWQJhHVk+yRyTt9RUrpsre8tyR4ZsMrqA3+/RPl2iwzfDiRArq
QDL2eEECgYEA3toHy0CDlcU+AfZxH0RY051385e/qrQPkH5rQ1vSriw7GjlJqKEd
qwH8t5U/BY0rnp2+t1tk91uhqtG0jVRf8C742rZDiXlGXf+pq7ZdiXr1JQBHUDoZ
vf2FL8Z20ZOKoy5CJ26qQiT87BwuL+GS7w+HzjYmCL39SE3QzxY+rs4=
-----END RSA PRIVATE KEY-----`

func TestWithAccountActivity(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	secrets := &Secrets{GooglePrivateKeySecret: dummyGoogleCert}
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
			secrets := &Secrets{GooglePrivateKeySecret: dummyGoogleCert}
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
