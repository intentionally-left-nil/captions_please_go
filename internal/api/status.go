package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
)

// indirection for unit tests
var newTwitter = twitter.NewTwitter

func WebhookStatus(ctx context.Context, req *http.Request) APIResponse {
	secrets := GetSecrets(ctx)
	twitter := newTwitter(secrets.TwitterConsumerKey,
		secrets.TwitterConsumerSecret,
		secrets.TwitterAccessToken,
		secrets.TwitterAccessTokenSecret,
		secrets.TwitterBearerToken)

	err := ensureWebhook(twitter, secrets.WebhookUrl)
	if err == nil {
		err = ensureSubscription(twitter)
	}

	if err == nil {
		return APIResponse{status: http.StatusOK}
	}

	return APIResponse{
		status: http.StatusInternalServerError,
	}
}

func ensureWebhook(twitter twitter.Twitter, webhookURL string) error {
	webhooks, err := twitter.GetWebhooks()
	if err != nil {
		logrus.Error(fmt.Sprintf("Unable to get the current webhook status %v", err))
		return err
	}

	if len(webhooks) == 1 && webhooks[0].Valid {
		return nil
	}

	logrus.Info("Webhook was not valid, trying to recreate it")

	for _, webhook := range webhooks {
		logrus.Info(fmt.Sprintf("Deleting webhook %v", loggify(webhook)))
		err = twitter.DeleteWebhook(webhook.Id)
		if err != nil {
			logrus.Error(fmt.Sprintf("Unable to delete the webhook, trying to continue %v", err))
		}
	}
	webhook, err := twitter.CreateWebhook(webhookURL)

	if err != nil {
		logrus.Error(fmt.Sprintf("Unable to create a new webhook %v", err))
		return err
	}

	logrus.Info(fmt.Sprintf("New webhook created %v", loggify(webhook)))
	return nil
}

func ensureSubscription(twitter twitter.Twitter) error {
	subscriptions, err := twitter.GetSubscriptions()
	if err != nil {
		return err
	}
	if len(subscriptions) == 1 {
		return nil
	}

	for _, subscription := range subscriptions {
		err = twitter.DeleteSubscription(subscription.Id)
		if err != nil {
			logrus.Error(fmt.Sprintf("Unable to delete the webhook, trying to continue %v", err))
		}
	}
	err = twitter.AddSubscription()
	return err
}

func loggify(v interface{}) string {
	message, _ := json.MarshalIndent(v, "", "  ")
	return string(message)
}
