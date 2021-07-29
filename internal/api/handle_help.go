package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
)

type helpJob struct {
	ctx        context.Context
	tweetId    string
	message    string
	onComplete context.CancelFunc
	out        chan<- ActivityResult
}

type helpKey int

const theHelpKey helpKey = 0

type helpState struct {
	config HelpConfig
	jobs   chan helpJob
}

func WithHelp(ctx context.Context, config HelpConfig, client twitter.Twitter) context.Context {
	validateHelpConfig(&config)
	state := &helpState{
		config: config,
		jobs:   make(chan helpJob, config.PendingHelpMessages),
	}
	ctx = context.WithValue(ctx, theHelpKey, state)

	// Create numWorkers goRoutines to read from the same (buffered) channel
	for i := 0; i < int(config.Workers); i++ {
		go func(i int) {
			logrus.Debug(fmt.Sprintf("Initializing Help worker %d", i))
			for job := range state.jobs {
				logrus.Debug(fmt.Sprintf("Worker %d processing tweet %s", i, job.tweetId))
				result := ActivityResult{action: "Reply with help message"}
				_, err := client.TweetReply(job.ctx, job.tweetId, job.message)
				result.err = err
				logrus.Debug(fmt.Sprintf("Worker %d finished processing tweet %s", i, job.tweetId))
				job.out <- result
				close(job.out)
			}
			logrus.Debug(fmt.Sprintf("Worker %d says bye-bye", i))
		}(i)
	}

	// When the parent context closes, tell the workers to exit
	go func() {
		<-ctx.Done()
		close(state.jobs)
	}()
	return ctx
}

func HandleHelp(ctx context.Context, tweet *twitter.Tweet, message string) <-chan ActivityResult {
	state := getHelpState(ctx)
	logrus.Debug(fmt.Sprintf("Help job %s waiting with timeout %d", tweet.Id, state.config.Timeout))
	out := make(chan ActivityResult)
	helpCtx, onComplete := context.WithTimeout(ctx, state.config.Timeout)
	hJob := helpJob{ctx: helpCtx, out: out, onComplete: onComplete, message: message, tweetId: tweet.Id}
	select {
	case state.jobs <- hJob:
		logrus.Debug(fmt.Sprintf("Help job enqueued successfully for tweet %s", tweet.Id))
	case <-helpCtx.Done():
		go func() {
			logrus.Info(fmt.Sprintf("Help job %s timed out before it could be picked up", tweet.Id))
			out <- ActivityResult{action: "enqueue help job", err: errors.New("timeout")}
			logrus.Debug(fmt.Sprintf("Sent timeout for %s", tweet.Id))
			close(out)
		}()
	}
	return out
}

func getHelpState(ctx context.Context) *helpState {
	return ctx.Value(theHelpKey).(*helpState)
}

func validateHelpConfig(config *HelpConfig) {
	if config.Timeout == 0 {
		config.Timeout = time.Second * 30
	}

	if config.Workers == 0 {
		config.Workers = 1
	}
}
