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

	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
)

type activityData struct {
	CreateData      []twitter.Tweet `json:"tweet_create_events"`
	FromBlockedUser bool            `json:"user_has_blocked"`
	UserId          string          `json:"source"`
	BotId           string          `json:"for_user_id"`
}

type HelpConfig struct {
	Workers             uint
	PendingHelpMessages uint
	Timeout             time.Duration
}

type ActivityConfig struct {
	Workers            uint
	MaxOutstandingJobs uint
	WebhookTimeout     time.Duration
	Help               HelpConfig
}

type ActivityResult struct {
	tweet  *twitter.Tweet
	action string
	err    error
}

type activityJob struct {
	botId string
	tweet *twitter.Tweet
	out   chan<- ActivityResult
}

type activityState struct {
	jobs   chan activityJob
	config ActivityConfig
}

type activityStateKey int

const theActivityStateKey activityStateKey = 0

func WithAccountActivity(ctx context.Context, config ActivityConfig, client twitter.Twitter) context.Context {
	validateActivityConfig(&config)
	logrus.Debug(fmt.Sprintf("Initializing AccountActivity with %d workers and %d outstanding jobs", config.Workers, config.MaxOutstandingJobs))
	state := &activityState{
		config: config,
		jobs:   make(chan activityJob, config.MaxOutstandingJobs),
	}
	ctx = context.WithValue(ctx, theActivityStateKey, state)
	ctx = WithHelp(ctx, config.Help, client)
	ctx = WithAltText(ctx, client)

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
	return ctx
}

func singleActivityResult(result ActivityResult) <-chan ActivityResult {
	logrus.Debug(fmt.Sprintf("Sending single action %s and err %v", result.action, result.err))
	// It's important to buffer this channel because we haven't returned the out channel to the caller
	// and therefore nobody is listening yet. Otherwise this will deadlock
	out := make(chan ActivityResult, 1)
	out <- result
	logrus.Debug(fmt.Sprintf("Sent single action %s and err %v", result.action, result.err))
	close(out)
	return out
}

func AccountActivityWebhook(ctx context.Context, req *http.Request) (APIResponse, <-chan ActivityResult) {
	data := activityData{}
	err := twitter.GetJSON(&http.Response{Body: req.Body, StatusCode: http.StatusOK}, &data)
	logDebugJSON(data)
	if err != nil {
		return APIResponse{status: http.StatusBadRequest}, singleActivityResult(ActivityResult{action: "parsing json", err: err})
	}

	if data.BotId == "" {
		return APIResponse{status: http.StatusBadRequest}, singleActivityResult(ActivityResult{action: "parsing json", err: errors.New("missing for_user_id")})
	}

	if data.FromBlockedUser {
		return APIResponse{status: http.StatusOK}, singleActivityResult(ActivityResult{action: "ignoring blocked user"})
	}

	if len(data.CreateData) == 0 {
		return APIResponse{status: http.StatusOK}, singleActivityResult(ActivityResult{action: "no creation events"})
	}

	// 1. Create a multiplexer to store the results for parsing each tweet
	combinedOut := make(chan ActivityResult)

	// 2. Create a wait group so we know when all the tweets have been processed & we can close the combinedOut multiplexer
	wg := sync.WaitGroup{}
	wg.Add(len(data.CreateData))

	state := getActivityState(ctx)
	for _, tweet := range data.CreateData {
		// 3. Make a goroutine for each tweet, create an out channel, and try to pass the job (tweet + out) to the thread pool
		// Once queued, this delegates ownership responsibility to the thread pool - it is responsible for filling the channel
		// AND closing it.
		// If we can't queue it due to backpressure, then we need to propagate the timeout upwards
		out := make(chan ActivityResult)
		tweet := tweet
		job := activityJob{botId: data.BotId, tweet: &tweet, out: out}
		go func() {
			select {
			case state.jobs <- job:
				logrus.Debug(fmt.Sprintf("Activity: Enqueued tweet %s", tweet.Id))

			case <-time.After(state.config.WebhookTimeout):
				logrus.Info(fmt.Sprintf("Job queue is backed up, dropping tweet %s", tweet.Id))
				result := ActivityResult{action: "enqueue activity job", err: errors.New("timeout")}
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

func handleNewTweetActivity(ctx context.Context, job activityJob) <-chan ActivityResult {
	botMention := getMention(job.botId, job.tweet)
	if botMention == nil || job.tweet.User.Id == job.botId {
		result := ActivityResult{action: "User didnt mention us. Ignoring"}
		out := make(chan ActivityResult)
		go func() {
			out <- result
			close(out)
		}()
		return out
	}
	command := getCommand(job.tweet, botMention)
	return handleCommand(ctx, command, job)
}

func getMention(botId string, tweet *twitter.Tweet) *twitter.Mention {
	for _, mention := range tweet.Mentions {
		if mention.Id == botId {
			return &mention
		}
	}
	return nil
}

func getCommand(tweet *twitter.Tweet, mention *twitter.Mention) string {
	command := ""
	if mention.EndIndex+1 <= len(tweet.Text) {
		command = strings.TrimSpace(tweet.Text[mention.EndIndex:])
	}

	if command == "" {
		command = "auto"
	}
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
