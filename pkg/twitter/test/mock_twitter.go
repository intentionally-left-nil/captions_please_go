package twitter_test

import (
	"context"
	"net/http"
	"testing"

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
	TweetReplyMock         func(tweetID string, message string) (*twitter.Tweet, error)
}

func (m *MockTwitter) GetWebhooks(ctx context.Context) ([]twitter.Webhook, twitter.RateLimit, error) {
	assert.NotNil(m.T, m.GetWebhooksMock)
	webhooks, err := m.GetWebhooksMock()
	return webhooks, twitter.RateLimit{}, err
}

func (m *MockTwitter) CreateWebhook(ctx context.Context, url string) (twitter.Webhook, twitter.RateLimit, error) {
	assert.NotNil(m.T, m.CreateWebhookMock)
	webhook, err := m.CreateWebhookMock(url)
	return webhook, twitter.RateLimit{}, err
}

func (m *MockTwitter) DeleteWebhook(ctx context.Context, id string) (twitter.RateLimit, error) {
	assert.NotNil(m.T, m.DeleteWebhookMock)
	return twitter.RateLimit{}, m.DeleteWebhookMock(id)
}

func (m *MockTwitter) GetSubscriptions(ctx context.Context) ([]twitter.Subscription, twitter.RateLimit, error) {
	assert.NotNil(m.T, m.GetSubscriptionsMock)
	subscription, err := m.GetSubscriptionsMock()
	return subscription, twitter.RateLimit{}, err
}

func (m *MockTwitter) DeleteSubscription(ctx context.Context, id string) (twitter.RateLimit, error) {
	assert.NotNil(m.T, m.DeleteSubscriptionMock)
	return twitter.RateLimit{}, m.DeleteSubscriptionMock(id)
}

func (m *MockTwitter) AddSubscription(ctx context.Context) (twitter.RateLimit, error) {
	assert.NotNil(m.T, m.AddSubscriptionMock)
	return twitter.RateLimit{}, m.AddSubscriptionMock()
}

func (m *MockTwitter) GetTweet(ctx context.Context, id string) (*twitter.Tweet, twitter.RateLimit, error) {
	assert.NotNil(m.T, m.GetTweetMock)
	tweet, err := m.GetTweetMock(id)
	return tweet, twitter.RateLimit{}, err
}

func (m *MockTwitter) GetTweetRaw(ctx context.Context, id string) (*http.Response, twitter.RateLimit, error) {
	assert.NotNil(m.T, m.GetTweetRawMock)
	resp, err := m.GetTweetRawMock(id)
	return resp, twitter.RateLimit{}, err
}

func (m *MockTwitter) TweetReply(ctx context.Context, id string, message string) (*twitter.Tweet, twitter.RateLimit, error) {
	assert.NotNil(m.T, m.TweetReplyMock)
	tweet, err := m.TweetReplyMock(id, message)
	return tweet, twitter.RateLimit{}, err
}
