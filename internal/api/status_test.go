package api

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/stretchr/testify/assert"
)

type mockTwitter struct {
	t                  *testing.T
	getWebhooks        func() ([]twitter.Webhook, error)
	createWebhook      func(string) (twitter.Webhook, error)
	deleteWebhook      func(string) error
	getSubscriptions   func() ([]twitter.Subscription, error)
	deleteSubscription func(string) error
	addSubscription    func() error
}

func (m *mockTwitter) GetWebhooks() ([]twitter.Webhook, error) {
	assert.NotNil(m.t, m.getWebhooks)
	return m.getWebhooks()
}

func (m *mockTwitter) CreateWebhook(url string) (twitter.Webhook, error) {
	assert.NotNil(m.t, m.createWebhook)
	return m.createWebhook(url)
}

func (m *mockTwitter) DeleteWebhook(id string) error {
	assert.NotNil(m.t, m.deleteWebhook)
	return m.deleteWebhook(id)
}

func (m *mockTwitter) GetSubscriptions() ([]twitter.Subscription, error) {
	assert.NotNil(m.t, m.getSubscriptions)
	return m.getSubscriptions()
}

func (m *mockTwitter) DeleteSubscription(id string) error {
	assert.NotNil(m.t, m.deleteSubscription)
	return m.deleteSubscription(id)
}

func (m *mockTwitter) AddSubscription() error {
	assert.NotNil(m.t, m.addSubscription)
	return m.addSubscription()
}

func (m *mockTwitter) GetTweet(id string) (*twitter.Tweet, error) {
	return nil, errors.New("Not implemented")
}

func TestWebhookStatus(t *testing.T) {

	err := errors.New("oops, I did it again")

	noWebhooks := func() ([]twitter.Webhook, error) {
		return []twitter.Webhook{}, nil
	}

	oneWebhook := func() ([]twitter.Webhook, error) {
		return []twitter.Webhook{{Valid: true}}, nil
	}

	noSubscriptions := func() ([]twitter.Subscription, error) {
		return []twitter.Subscription{}, nil
	}
	oneSubscription := func() ([]twitter.Subscription, error) {
		return []twitter.Subscription{{Id: "123"}}, nil
	}

	addValidSubscription := func() error { return nil }

	createValidWebhook := func(s string) (twitter.Webhook, error) { return twitter.Webhook{}, nil }

	tests := []struct {
		name               string
		getWebhooks        func() ([]twitter.Webhook, error)
		createWebhook      func(string) (twitter.Webhook, error)
		deleteWebhook      func(string) error
		getSubscriptions   func() ([]twitter.Subscription, error)
		addSubscription    func() error
		deleteSubscription func(string) error
		statusCode         int
	}{
		{
			name:             "Succeeds if the webhook is already valid",
			statusCode:       200,
			getWebhooks:      oneWebhook,
			getSubscriptions: oneSubscription,
		},
		{
			name:             "Creates a new webhook if none are present",
			statusCode:       200,
			getWebhooks:      noWebhooks,
			createWebhook:    createValidWebhook,
			getSubscriptions: noSubscriptions,
			addSubscription:  addValidSubscription,
		},
		{
			name:       "Creates a new webhook if the existing one is invalid",
			statusCode: 200,
			getWebhooks: func() ([]twitter.Webhook, error) {
				return []twitter.Webhook{{Valid: false}}, nil
			},
			deleteWebhook:    func(s string) error { return nil },
			createWebhook:    createValidWebhook,
			getSubscriptions: noSubscriptions,
			addSubscription:  addValidSubscription,
		},
		{
			name:        "Regenerates the subscription of an existing webhook",
			statusCode:  200,
			getWebhooks: oneWebhook,
			getSubscriptions: func() ([]twitter.Subscription, error) {
				return []twitter.Subscription{}, nil
			},
			addSubscription: func() error { return nil },
		},
		{
			name:             "Handles multiple webhooks by deleting all of them",
			statusCode:       200,
			getWebhooks:      func() ([]twitter.Webhook, error) { return []twitter.Webhook{{}, {}}, nil },
			deleteWebhook:    func(s string) error { return nil },
			createWebhook:    createValidWebhook,
			getSubscriptions: noSubscriptions,
			addSubscription:  addValidSubscription,
		},
		{
			name:             "Ignores webhook deletion errors and tries to continue",
			statusCode:       200,
			getWebhooks:      func() ([]twitter.Webhook, error) { return []twitter.Webhook{{}, {}}, nil },
			deleteWebhook:    func(s string) error { return err },
			createWebhook:    createValidWebhook,
			getSubscriptions: noSubscriptions,
			addSubscription:  addValidSubscription,
		},
		{
			name:        "Handles multiple subscriptions by deleting them",
			statusCode:  200,
			getWebhooks: oneWebhook,
			getSubscriptions: func() ([]twitter.Subscription, error) {
				return []twitter.Subscription{{}, {}}, nil
			},
			deleteSubscription: func(string) error {
				return nil
			},
			addSubscription: addValidSubscription,
		},
		{
			name:        "Ignores subscription deletion failures",
			statusCode:  200,
			getWebhooks: oneWebhook,
			getSubscriptions: func() ([]twitter.Subscription, error) {
				return []twitter.Subscription{{}, {}}, nil
			},
			deleteSubscription: func(string) error {
				return err
			},
			addSubscription: addValidSubscription,
		},
		{
			name:       "errors if getWebhooks fails",
			statusCode: 500,
			getWebhooks: func() ([]twitter.Webhook, error) {
				return nil, err
			},
		},
		{
			name:             "errors if getSubscriptions fails",
			statusCode:       500,
			getWebhooks:      oneWebhook,
			getSubscriptions: func() ([]twitter.Subscription, error) { return nil, err },
		},
		{
			name:          "errors if createWebhook fails",
			statusCode:    500,
			getWebhooks:   noWebhooks,
			createWebhook: func(s string) (twitter.Webhook, error) { return twitter.Webhook{}, err },
		},
		{
			name:             "errors if addSubscription fails",
			statusCode:       500,
			getWebhooks:      oneWebhook,
			getSubscriptions: noSubscriptions,
			addSubscription:  func() error { return err },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := &http.Request{}
			secrets := &Secrets{}
			ctx := withSecrets(context.Background(), secrets)

			originalNewTwitter := newTwitter
			newTwitter = func(_ string, _ string, _ string, _ string, _ string) twitter.Twitter {
				return &mockTwitter{
					t:                  t,
					getWebhooks:        test.getWebhooks,
					createWebhook:      test.createWebhook,
					deleteWebhook:      test.deleteWebhook,
					getSubscriptions:   test.getSubscriptions,
					addSubscription:    test.addSubscription,
					deleteSubscription: test.deleteSubscription,
				}
			}
			defer func() {
				newTwitter = originalNewTwitter
			}()

			response := WebhookStatus(ctx, req)
			assert.Equal(t, test.statusCode, response.status)
		})

	}
}
