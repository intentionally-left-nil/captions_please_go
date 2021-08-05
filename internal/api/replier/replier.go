package replier

import (
	"context"
	"fmt"

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
}
type replierCtxKey int

const theReplierKey replierCtxKey = 0

func WithReplier(ctx context.Context, client twitter.Twitter) (context.Context, error) {
	err := message.LoadMessages()
	if err == nil {
		state := &replierState{client: client}
		ctx = setReplierState(ctx, state)
	}
	return ctx, err
}

func Reply(ctx context.Context, tweet *twitter.Tweet, message message.Localized) ReplyResult {
	logrus.Debug(fmt.Sprintf("%s Reply called with %s", tweet.Id, message))
	remaining, err := splitMessage(string(message))
	if err != nil {
		return ReplyResult{Err: err, ParentTweet: tweet}
	}
	state := getReplierState(ctx)
	return replyHelper(ctx, state.client, tweet, remaining)
}

func replyHelper(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet, remaining []string) ReplyResult {
	if len(remaining) == 0 {
		return ReplyResult{ParentTweet: tweet}
	}
	nextTweet, err := client.TweetReply(ctx, tweet.Id, remaining[0])
	if err != nil && err.Type() == structured_error.TweetTooLong {
		// The twitter-text library we use isn't fully up to date and gets in wrong sometimes
		// As a fallback just cut the tweet in half and try to send it
		logrus.Error(fmt.Sprintf("%s: The reply was too long: %s", tweet.Id, remaining[0]))
		first, second := splitInTwo(remaining[0])
		logrus.Debug(fmt.Sprintf("%s: Trying to send the smaller tweet %s", tweet.Id, first))
		nextTweet, err = client.TweetReply(ctx, tweet.Id, first)
		if err == nil {
			logrus.Debug(fmt.Sprintf("%s: Succeeded sending the smaller tweet", tweet.Id))
			// We were successful, so convert remaining from [tooLong, nextTweet...]
			// into [first, second, nextTweet, ...]
			// so we don't lose the remainder
			remaining = append([]string{first, second}, remaining[1:]...)
		}
	}
	if err != nil {
		return ReplyResult{Err: err, ParentTweet: tweet, Remaining: remaining}
	}
	return replyHelper(ctx, client, nextTweet, remaining[1:])
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
