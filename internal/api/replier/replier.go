package replier

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
)

type ReplyResult struct {
	ParentTweet *twitter.Tweet
	Remaining   []string
	Err         structured_error.StructuredError
}

type replierState struct {
	client twitter.Twitter
	dryRun bool
}
type replierCtxKey int

const theReplierKey replierCtxKey = 0

var after func(time.Duration) <-chan time.Time = time.After

func WithReplier(ctx context.Context, client twitter.Twitter, dryRun bool) (context.Context, error) {
	err := message.LoadMessages()
	if err == nil {
		state := &replierState{client: client, dryRun: dryRun}
		ctx = setReplierState(ctx, state)
	}
	return ctx, err
}

func Reply(ctx context.Context, tweet *twitter.Tweet, message message.Localized) (result ReplyResult) {
	logrus.Debug(fmt.Sprintf("%s Reply called with %s", tweet.Id, message))
	remaining, err := splitMessage(string(message))
	if err != nil {
		return ReplyResult{Err: err, ParentTweet: tweet}
	}
	state := getReplierState(ctx)
	if state.dryRun {
		fmt.Println("DRY RUN WOULD HAVE TWEETED THE FOLLOWING TWEETS:")
		for _, message := range remaining {
			fmt.Println(message)
		}
		fmt.Println("END DRY RUN")
		result = ReplyResult{
			ParentTweet: tweet,
		}
	} else {
		result = replyHelper(ctx, state.client, tweet, remaining)
	}
	return result
}

func replyHelper(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet, remaining []string) ReplyResult {
	if len(remaining) == 0 {
		return ReplyResult{ParentTweet: tweet}
	}
	nextTweet, err := client.TweetReply(ctx, tweet, remaining[0])
	if err != nil && err.Type() == structured_error.TweetTooLong {
		// The twitter-text library we use isn't fully up to date and gets in wrong sometimes
		// As a fallback just cut the tweet in half and try to send it
		logrus.Error(fmt.Sprintf("%s: The reply was too long: %s", tweet.Id, remaining[0]))
		first, second := splitInTwo(remaining[0])
		logrus.Debug(fmt.Sprintf("%s: Trying to send the smaller tweet %s", tweet.Id, first))
		nextTweet, err = client.TweetReply(ctx, tweet, first)
		if err == nil {
			logrus.Debug(fmt.Sprintf("%s: Succeeded sending the smaller tweet", tweet.Id))
			// We were successful, so convert remaining from [tooLong, nextTweet...]
			// into [first, second, nextTweet, ...]
			// so we don't lose the remainder
			remaining = append([]string{first, second}, remaining[1:]...)
		}
	}

	if err != nil && err.Type() == structured_error.CaseOfTheMissingTweet {
		// Sometimes, inexplicably we get this error in the middle of replying with a chain of tweets
		// Best working theory is that twitter needs some time to catch-up to the tweets being created, so
		// we'll wait and try one more time
		select {
		case <-ctx.Done():
			// do nothing if the context gets closed before we're done waiting
			logrus.Debug(fmt.Sprintf("%s: timeout before retrying CaseOfTheMissingTweet", tweet.Id))
		case <-after(time.Second * 30):
			logrus.Debug(fmt.Sprintf("%s retrying reply", tweet.Id))
			nextTweet, err = client.TweetReply(ctx, tweet, remaining[0])
			if err != nil && err.Type() == structured_error.DuplicateTweet {
				// Twitter is really having trouble with their API
				// Sometimes, we get the following behavior: The first tweet returns CaseOfTheMissingTweet
				// but... actually it suceeds. Then, after we wait 30 seconds and try again
				// now twitter is: Actually that tweet exists. So, now we have to go find it
				// because the first attempt returned an error, not the new tweet Id.
				logrus.Debug(fmt.Sprintf("%s: First CaseOfTheMissingTweet, now duplicate tweet", tweet.Id))
				nextTweet, err = findMissingReply(ctx, client, tweet.Id, remaining[0])
				if err == nil {
					logrus.Debug(fmt.Sprintf("%s Found the formerly missing, and now duplicate tweet %v", tweet.Id, nextTweet))
				}
			}
		}
	}

	if err != nil {
		return ReplyResult{Err: err, ParentTweet: tweet, Remaining: remaining}
	}
	return replyHelper(ctx, client, nextTweet, remaining[1:])
}

func findMissingReply(ctx context.Context, client twitter.Twitter, parentTweetId string, text string) (*twitter.Tweet, structured_error.StructuredError) {
	timelineTweets, err := client.UserTimeline(ctx, "captions_please", parentTweetId)
	if err != nil {
		return nil, err
	}
	for _, timelineTweet := range timelineTweets {
		if timelineTweet.ParentTweetId == parentTweetId {
			if timelineTweet.VisibleText == text {
				return timelineTweet, nil
			} else {
				logrus.Debug(fmt.Sprintf("%s: TimelineTweet has the correct parent, but the text doesn't match: %v", parentTweetId, timelineTweet))
			}
		}
	}
	return nil, structured_error.Wrap(errors.New("timeline tweet not found"), structured_error.TweetNotFound)
}

func setReplierState(ctx context.Context, state *replierState) context.Context {
	return context.WithValue(ctx, theReplierKey, state)
}

func getReplierState(ctx context.Context) *replierState {
	return ctx.Value(theReplierKey).(*replierState)
}

func splitInTwo(message string) (string, string) {
	runes := []rune(message)
	midpoint := len(runes) / 2
	first := runes[:midpoint]
	second := runes[midpoint:]
	return string(first), string(second)
}
