package api

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	twitter_test "github.com/AnilRedshift/captions_please_go/pkg/twitter/test"
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
)

func TestWithHelp(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = WithHelp(ctx, HelpConfig{})
	state := getHelpState(ctx)
	assert.NotNil(t, state)
}

func TestHandleHelp(t *testing.T) {
	// To help debugging errors, turn this on
	// logrus.SetLevel(logrus.DebugLevel)
	anError := errors.New("no help here")
	tests := []struct {
		name            string
		messages        []string
		timesToDelay    int
		twitterErr      error
		numErrors       int
		expectedActions []string
	}{
		{
			name:            "Replies to a tweet when the queue is empty",
			messages:        []string{"hello test"},
			expectedActions: []string{"Reply with help message"},
		},
		{

			name:            "Propagates the error upstream",
			messages:        []string{"hello test"},
			twitterErr:      anError,
			numErrors:       1,
			expectedActions: []string{"Reply with help message"},
		},
		{
			name:            "Times out when waiting for an open slot",
			messages:        []string{"first", "second"},
			timesToDelay:    1,
			numErrors:       1,
			expectedActions: []string{"Reply with help message"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer leaktest.Check(t)()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var delayCount uint64
			tweetReplyMock := func(id string, message string) (*twitter.Tweet, error) {
				tweet := twitter.Tweet{}
				count := atomic.AddUint64(&delayCount, 1)
				if count <= uint64(test.timesToDelay) {
					time.Sleep(time.Millisecond * 100)
				}
				return &tweet, test.twitterErr
			}
			mockTwitter := &twitter_test.MockTwitter{TweetReplyMock: tweetReplyMock}
			config := HelpConfig{Workers: 1, Timeout: time.Millisecond * 50, PendingHelpMessages: 0}
			ctx, err := replier.WithReplier(ctx, mockTwitter)
			assert.NoError(t, err)
			ctx = WithHelp(ctx, config)

			outs := make([]<-chan ActivityResult, len(test.messages))
			wg := sync.WaitGroup{}
			wg.Add(len(test.messages))

			for i, message := range test.messages {
				i := i
				message := message
				tweet := twitter.Tweet{Id: strconv.FormatInt(int64(i), 10)}
				go func(message string) {
					outs[i] = HandleHelp(ctx, &tweet, message)
					wg.Done()

				}(message)
			}
			wg.Wait()
			results := []ActivityResult{}
			for _, out := range outs {
				for result := range out {
					results = append(results, result)

				}
			}
			assert.Equal(t, len(test.messages), len(results))
			actions := results
			for _, expectedAction := range test.expectedActions {
				found := false
				for i, result := range actions {
					if result.action == expectedAction {
						// remove it from the slice
						actions[i] = actions[len(actions)-1]
						actions = actions[:len(actions)-1]
						found = true
						break
					}
				}
				assert.True(t, found, "Looking for action %s", expectedAction)
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
