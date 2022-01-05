package twitter_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/stretchr/testify/assert"
)

type MockTwitter struct {
	T                      *testing.T
	GetWebhooksMock        func() ([]twitter.Webhook, error)
	CreateWebhookMock      func(string) (twitter.Webhook, error)
	DeleteWebhookMock      func(string) error
	GetSubscriptionsMock   func() ([]twitter.Subscription, error)
	DeleteSubscriptionMock func(string) error
	AddSubscriptionMock    func() error
	GetTweetMock           func(tweetID string) (*twitter.Tweet, error)
	GetTweetRawMock        func(tweetID string) (*http.Response, error)
	TweetReplyMock         func(tweet *twitter.Tweet, message string) (*twitter.Tweet, error)
}

func (m *MockTwitter) GetWebhooks(ctx context.Context) ([]twitter.Webhook, structured_error.StructuredError) {
	assert.NotNil(m.T, m.GetWebhooksMock)
	webhooks, err := m.GetWebhooksMock()
	return webhooks, structured_error.Wrap(err, structured_error.TwitterError)
}

func (m *MockTwitter) CreateWebhook(ctx context.Context, url string) (twitter.Webhook, structured_error.StructuredError) {
	assert.NotNil(m.T, m.CreateWebhookMock)
	webhook, err := m.CreateWebhookMock(url)
	return webhook, structured_error.Wrap(err, structured_error.TwitterError)
}

func (m *MockTwitter) DeleteWebhook(ctx context.Context, id string) structured_error.StructuredError {
	assert.NotNil(m.T, m.DeleteWebhookMock)
	err := m.DeleteWebhookMock(id)
	return structured_error.Wrap(err, structured_error.TwitterError)
}

func (m *MockTwitter) GetSubscriptions(ctx context.Context) ([]twitter.Subscription, structured_error.StructuredError) {
	assert.NotNil(m.T, m.GetSubscriptionsMock)
	subscription, err := m.GetSubscriptionsMock()
	return subscription, structured_error.Wrap(err, structured_error.TwitterError)
}

func (m *MockTwitter) DeleteSubscription(ctx context.Context, id string) structured_error.StructuredError {
	assert.NotNil(m.T, m.DeleteSubscriptionMock)
	err := m.DeleteSubscriptionMock(id)
	return structured_error.Wrap(err, structured_error.TwitterError)
}

func (m *MockTwitter) AddSubscription(ctx context.Context) structured_error.StructuredError {
	assert.NotNil(m.T, m.AddSubscriptionMock)
	err := m.AddSubscriptionMock()
	return structured_error.Wrap(err, structured_error.TwitterError)
}

func (m *MockTwitter) GetTweet(ctx context.Context, id string) (*twitter.Tweet, structured_error.StructuredError) {
	assert.NotNil(m.T, m.GetTweetMock)
	tweet, err := m.GetTweetMock(id)
	return tweet, structured_error.Wrap(err, structured_error.TwitterError)
}

func (m *MockTwitter) GetTweetRaw(ctx context.Context, id string) (*http.Response, structured_error.StructuredError) {
	assert.NotNil(m.T, m.GetTweetRawMock)
	resp, err := m.GetTweetRawMock(id)
	return resp, structured_error.Wrap(err, structured_error.TwitterError)
}

func (m *MockTwitter) TweetReply(ctx context.Context, tweet *twitter.Tweet, message string) (*twitter.Tweet, structured_error.StructuredError) {
	assert.NotNil(m.T, m.TweetReplyMock)
	tweet, err := m.TweetReplyMock(tweet, message)
	return tweet, structured_error.Wrap(err, structured_error.TwitterError)
}
