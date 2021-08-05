package replier

import (
	"context"

	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
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
