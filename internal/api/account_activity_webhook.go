package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
)

type activityData struct {
	CreateData      []twitter.Tweet `json:"tweet_create_events"`
	FromBlockedUser bool            `json:"user_has_blocked"`
	UserId          string          `json:"source"`
	BotId           string          `json:"for_user_id"`
}

type ActivityConfig struct {
	Workers            uint
	MaxOutstandingJobs uint
	WebhookTimeout     time.Duration
}

type activityJob struct {
	botId string
	tweet *twitter.Tweet
	out   chan<- common.ActivityResult
}

type activityState struct {
	jobs   chan activityJob
	config ActivityConfig
}

type activityStateKey int

const theActivityStateKey activityStateKey = 0

func WithAccountActivity(ctx context.Context, config ActivityConfig, client twitter.Twitter) (context.Context, error) {
	var err error
	validateActivityConfig(&config)
	logrus.Debug(fmt.Sprintf("Initializing AccountActivity with %d workers and %d outstanding jobs", config.Workers, config.MaxOutstandingJobs))
	state := &activityState{
		config: config,
		jobs:   make(chan activityJob, config.MaxOutstandingJobs),
	}
	ctx = context.WithValue(ctx, theActivityStateKey, state)
	ctx = WithAltText(ctx, client)
	ctx = WithDescribe(ctx, client)
	ctx = WithAuto(ctx, client)
	ctx, err = WithOCR(ctx, client)
	if err == nil {
		ctx, err = replier.WithReplier(ctx, client)
	}

	if err == nil {
		for i := 0; i < int(config.Workers); i++ {
			go func(i int) {
				logrus.Debug(fmt.Sprintf("Initializing Activity worker %d", i))
				for job := range state.jobs {
					logrus.Debug(fmt.Sprintf("Worker %d processing job %s", i, job.tweet.Id))
					for result := range handleNewTweetActivity(ctx, job) {
						job.out <- result
					}
					close(job.out)
				}
			}(i)
		}

		go func() {
			<-ctx.Done()
			close(state.jobs)
		}()
	}
	return ctx, err
}

func singleActivityResult(result common.ActivityResult) <-chan common.ActivityResult {
	logrus.Debug(fmt.Sprintf("Sending single action %s and err %v", result.Action, result.Err))
	// It's important to buffer this channel because we haven't returned the out channel to the caller
	// and therefore nobody is listening yet. Otherwise this will deadlock
	out := make(chan common.ActivityResult, 1)
	out <- result
	logrus.Debug(fmt.Sprintf("Sent single action %s and err %v", result.Action, result.Err))
	close(out)
	return out
}

func AccountActivityWebhook(ctx context.Context, req *http.Request) (APIResponse, <-chan common.ActivityResult) {
	data := activityData{}
	err := twitter.GetJSON(&http.Response{Body: req.Body, StatusCode: http.StatusOK}, &data)
	logDebugJSON(data)
	if err != nil {
		return APIResponse{status: http.StatusBadRequest}, singleActivityResult(common.ActivityResult{Action: "parsing json", Err: err})
	}

	if data.BotId == "" {
		return APIResponse{status: http.StatusBadRequest}, singleActivityResult(common.ActivityResult{Action: "parsing json", Err: errors.New("missing for_user_id")})
	}

	if data.FromBlockedUser {
		return APIResponse{status: http.StatusOK}, singleActivityResult(common.ActivityResult{Action: "ignoring blocked user"})
	}

	if len(data.CreateData) == 0 {
		return APIResponse{status: http.StatusOK}, singleActivityResult(common.ActivityResult{Action: "no creation events"})
	}

	// 1. Create a multiplexer to store the results for parsing each tweet
	combinedOut := make(chan common.ActivityResult)

	// 2. Create a wait group so we know when all the tweets have been processed & we can close the combinedOut multiplexer
	wg := sync.WaitGroup{}
	wg.Add(len(data.CreateData))

	state := getActivityState(ctx)
	for _, tweet := range data.CreateData {
		// 3. Make a goroutine for each tweet, create an out channel, and try to pass the job (tweet + out) to the thread pool
		// Once queued, this delegates ownership responsibility to the thread pool - it is responsible for filling the channel
		// AND closing it.
		// If we can't queue it due to backpressure, then we need to propagate the timeout upwards
		out := make(chan common.ActivityResult)
		tweet := tweet
		job := activityJob{botId: data.BotId, tweet: &tweet, out: out}
		go func() {
			select {
			case state.jobs <- job:
				logrus.Debug(fmt.Sprintf("Activity: Enqueued tweet %s", tweet.Id))

			case <-time.After(state.config.WebhookTimeout):
				logrus.Info(fmt.Sprintf("Job queue is backed up, dropping tweet %s", tweet.Id))
				result := common.ActivityResult{Action: "enqueue activity job", Err: errors.New("timeout")}
				out <- result
				close(out)
			}
		}()

		// 4. Start another goroutine per-tweet to power the multiplexer & forward the result to combinedOut
		go func() {
			for result := range out {
				combinedOut <- result
			}
			// 5. We can't close combinedOut when done because other goroutines are writing to it
			// Hence the wait group serves as a side-channel notification to notify this routine is complete
			wg.Done()
		}()
	}

	go func() {
		// 6. Wait for all the goroutines to finish transfering results to combinedOut
		// Now we can close the combinedOut channel and let callers know we're done
		wg.Wait()
		close(combinedOut)
	}()
	return APIResponse{status: http.StatusOK}, combinedOut
}

func getActivityState(ctx context.Context) *activityState {
	return ctx.Value(theActivityStateKey).(*activityState)
}

func handleNewTweetActivity(ctx context.Context, job activityJob) <-chan common.ActivityResult {
	botMention := getVisibleMention(job.botId, job.tweet)
	if botMention == nil || job.tweet.User.Id == job.botId {
		result := common.ActivityResult{Action: "User didnt mention us. Ignoring"}
		out := make(chan common.ActivityResult)
		go func() {
			out <- result
			close(out)
		}()
		return out
	}
	command := getCommand(job.tweet, botMention)
	return handleCommand(ctx, command, job)
}

func getVisibleMention(botId string, tweet *twitter.Tweet) *twitter.Mention {
	for _, mention := range tweet.Mentions {
		if mention.Id == botId && mention.Visible {
			return &mention
		}
	}
	return nil
}

func getCommand(tweet *twitter.Tweet, mention *twitter.Mention) string {
	command := ""
	if mention.EndIndex+1 <= len(tweet.FullText) {
		command = strings.TrimSpace(tweet.FullText[mention.EndIndex+1:])
	}

	if command == "" {
		command = "auto"
	}
	logrus.Debug(fmt.Sprintf("command to parse is %s", command))
	return command
}

func logDebugJSON(v interface{}) {
	logrus.DebugFn(func() []interface{} {
		bytes, err := json.Marshal(v)
		if err == nil {
			return []interface{}{string(bytes)}
		}
		return []interface{}{err.Error()}
	})
}

func validateActivityConfig(config *ActivityConfig) {
	if config.WebhookTimeout == 0 {
		config.WebhookTimeout = time.Second * 30
	}

	if config.Workers == 0 {
		config.Workers = 1
	}
}
