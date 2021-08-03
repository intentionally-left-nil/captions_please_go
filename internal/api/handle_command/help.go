package handle_command

import (
	"context"
	"fmt"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
)

func Help(ctx context.Context, tweet *twitter.Tweet) <-chan common.ActivityResult {
	result := replier.Reply(ctx, tweet, replier.HelpMessage(language.English))
	if result.Err != nil {
		logrus.Info(fmt.Sprintf("%s: Replying with the help message failed with %v", tweet.Id, result.Err))
	}
	out := make(chan common.ActivityResult, 1)
	// Just drop help messages on the floor if there's an error. Never return an error upstream
	out <- common.ActivityResult{Tweet: tweet, Action: "reply with help"}
	close(out)
	return out
}
