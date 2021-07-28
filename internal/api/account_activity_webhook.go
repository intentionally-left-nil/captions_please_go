package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
)

type activityData struct {
	CreateData      *[]twitter.Tweet `json:"tweet_create_events"`
	FromBlockedUser bool             `json:"user_has_blocked"`
	UserId          string           `json:"source"`
	BotId           string           `json:"for_user_id"`
}

func AccountActivityWebhook(ctx context.Context, req *http.Request) APIResponse {
	data := activityData{}
	err := twitter.GetJSON(&http.Response{Body: req.Body, StatusCode: http.StatusOK}, &data)
	logDebugJSON(data)
	if err == nil {
		if data.FromBlockedUser {
			logrus.Info(fmt.Sprintf("ignoring webhook activity from blocked user %s", data.UserId))
			return APIResponse{status: http.StatusOK}
		}

		if data.CreateData != nil {
			for _, tweet := range *data.CreateData {
				activityErr := handleNewTweetActivity(data.BotId, &tweet)
				if activityErr != nil {
					logrus.Error(fmt.Sprintf("AccountActivityWebhook encountered an error %v with payload %v", activityErr, data))
					// Keep going, try to process the remaining tweets
				}
			}
		}
	}
	// We always tell twitter we're happy - especially since they don't retry anyways
	if err != nil {
		logrus.Error(fmt.Sprintf("AccountActivityWebhook encountered an error %v with payload %v", err, data))
	}
	return APIResponse{status: http.StatusOK}
}

func handleNewTweetActivity(botId string, tweet *twitter.Tweet) error {
	botMention := getMention(botId, tweet)
	if botMention == nil || tweet.User.Id == botId {
		logrus.Debug("Received a creation event where a user didn't mention us. Ignoring")
		return nil
	}
	command := getCommand(tweet, botMention)
	return handleCommand(command, tweet)
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
