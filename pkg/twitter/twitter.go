package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/mrjones/oauth"
	"github.com/sirupsen/logrus"
)

type twitter struct {
	client  *http.Client
	bearer  string
	limiter twitterLimiter
}

type RateLimit struct {
	Ceiling    *int
	Remaining  *int
	NextWindow *time.Time
}

func (r RateLimit) String() string {
	ceiling, remaining, nextWindow := "nil", "nil", "nil"
	if r.Ceiling != nil {
		ceiling = fmt.Sprintf("%d", *r.Ceiling)
	}

	if r.Remaining != nil {
		remaining = fmt.Sprintf("%d", *r.Remaining)
	}

	if r.NextWindow != nil {
		nextWindow = r.NextWindow.String()
	}
	return fmt.Sprintf("ceiling: %s, remaining %s, nextWindow %v", ceiling, remaining, nextWindow)
}

type twitterLimiter struct {
	lock   sync.RWMutex
	limits map[string]RateLimit
}

func (tl *twitterLimiter) getLimit(route string) RateLimit {
	tl.lock.RLock()
	defer tl.lock.RUnlock()
	return tl.limits[route]
}

func (tl *twitterLimiter) setLimit(route string, response *http.Response) {
	if response != nil {
		limit := getRateLimit(response)
		logrus.Debug(fmt.Sprintf("route %s received RateLimit %v", route, limit))
		existing := tl.getLimit(route)
		if existing.Remaining != nil && *existing.Remaining == 0 {
			tl.lock.Lock()
			defer tl.lock.Unlock()
			tl.limits[route] = limit
		}
	}
}

func (tl *twitterLimiter) wait(ctx context.Context, route string) error {
	limit := tl.getLimit(route)
	var duration time.Duration = 0
	if limit.Remaining != nil && *limit.Remaining == 0 {
		if limit.NextWindow == nil {
			// If twitter doesn't tell us when to try again, we'll give them 30 seconds
			duration = time.Second * 30
		} else {
			duration = time.Until(*limit.NextWindow)
		}
	}
	select {
	case <-time.After(duration):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout %v on route %s", route, limit)
	}
}

type Twitter interface {
	GetWebhooks(ctx context.Context) ([]Webhook, error)
	CreateWebhook(ctx context.Context, url string) (Webhook, error)
	DeleteWebhook(ctx context.Context, webhookID string) error
	GetSubscriptions(ctx context.Context) ([]Subscription, error)
	DeleteSubscription(ctx context.Context, subscriptionID string) error
	AddSubscription(ctx context.Context) error
	GetTweetRaw(ctx context.Context, tweetID string) (*http.Response, error)
	GetTweet(ctx context.Context, tweetID string) (*Tweet, error)
	TweetReply(ctx context.Context, tweetID string, message string) (*Tweet, structured_error.StructuredError)
}

type Webhook struct {
	Id    string `json:"id"`
	Url   string `json:"url"`
	Valid bool   `json:"valid"`
}

type Subscription struct {
	Id string `json:"user_id"`
}

const URL = "https://api.twitter.com/1.1/"

func NewTwitter(consumerKey string, consumerSecret string, accessToken string, accessTokenSecret string, bearerToken string) Twitter {
	c := oauth.NewConsumer(consumerKey, consumerSecret, oauth.ServiceProvider{})
	token := oauth.AccessToken{Token: accessToken, Secret: accessTokenSecret}
	client, _ := c.MakeHttpClient(&token)
	return &twitter{client: client, bearer: bearerToken}
}

func (t *twitter) GetWebhooks(ctx context.Context) ([]Webhook, error) {
	var webhooks []Webhook
	response, err := t.get(ctx, "get_webhooks", URL+"account_activity/all/dev/webhooks.json")
	if err == nil {
		webhooks = make([]Webhook, 0)
		err = GetJSON(response, &webhooks)
	}
	return webhooks, err
}

func (t *twitter) DeleteWebhook(ctx context.Context, webhookID string) error {
	err := t.limiter.wait(ctx, "delete_webhook")
	url := fmt.Sprintf("%saccount_activity/all/dev/webhooks/%s.json", URL, webhookID)
	if err == nil {
		var request *http.Request
		var response *http.Response
		request, err = http.NewRequestWithContext(ctx, "DELETE", url, nil)
		if err != nil {
			response, err = t.client.Do(request)
			t.limiter.setLimit("delete_webhook", response)

			if err == nil {
				var body []byte
				body, err = ioutil.ReadAll(response.Body)
				logrus.Debug(fmt.Sprintf("Twitter response:\n%v\n", string(body)))
				if err == nil {
					err = validateStatusCode(response)
				}
			}
		}
	}
	return err
}

func (t *twitter) CreateWebhook(ctx context.Context, webhookUrl string) (Webhook, error) {
	var webhook Webhook
	response, err := t.post(ctx, "create_webhook", URL+"account_activity/all/dev/webhooks.json", url.Values{"url": []string{webhookUrl}})
	if err == nil {
		err = GetJSON(response, &webhook)
	}
	return webhook, err
}

func (t *twitter) GetSubscriptions(ctx context.Context) ([]Subscription, error) {
	type apiResponse struct {
		Subscriptions []Subscription `json:"subscriptions"`
	}
	var subscriptions []Subscription
	err := t.limiter.wait(ctx, "get_subscriptions")
	if err == nil {
		var request *http.Request
		request, err = http.NewRequestWithContext(ctx, "GET", URL+"account_activity/all/dev/subscriptions/list.json", nil)
		if err == nil {
			request.Header.Set("Authorization", "Bearer "+t.bearer)
			client := http.Client{}
			var response *http.Response
			response, err = client.Do(request)
			t.limiter.setLimit("get_subscriptions", response)
			if err == nil {
				api := apiResponse{}
				err = GetJSON(response, &api)
				if err == nil {
					subscriptions = api.Subscriptions
				}
			}
		}
	}
	return subscriptions, err
}

func (t *twitter) DeleteSubscription(ctx context.Context, subscriptionID string) error {
	err := t.limiter.wait(ctx, "delete_subscription")
	if err == nil {
		url := fmt.Sprintf("%saccount_activity/all/dev/subscriptions/%s.json", URL, subscriptionID)
		var request *http.Request
		request, err = http.NewRequestWithContext(ctx, "DELETE", url, nil)
		if err == nil {
			request.Header.Set("Authorization", "Bearer "+t.bearer)
			client := http.Client{}
			var response *http.Response
			response, err = client.Do(request)
			t.limiter.setLimit("delete_subscription", response)
			if err == nil {
				var body []byte
				body, err = ioutil.ReadAll(response.Body)
				logrus.Debug(fmt.Sprintf("Twitter response:\n%v\n", string(body)))
				if err == nil {
					err = validateStatusCode(response)
				}
			}
		}
	}
	return err
}

func (t *twitter) AddSubscription(ctx context.Context) error {
	response, err := t.post(ctx, "add_subscription", URL+"account_activity/all/dev/subscriptions.json", url.Values{})
	if err == nil {
		var body []byte
		body, err = ioutil.ReadAll(response.Body)
		logrus.Debug(fmt.Sprintf("Twitter response:\n%v\n", string(body)))
		if err == nil {
			err = validateStatusCode(response)
		}
	}
	return err
}
func (t *twitter) GetTweet(ctx context.Context, tweetID string) (*Tweet, error) {
	tweet := Tweet{}
	response, err := t.GetTweetRaw(ctx, tweetID)
	if err == nil {
		err = GetJSON(response, &tweet)
	}
	return &tweet, err
}

func (t *twitter) GetTweetRaw(ctx context.Context, tweetID string) (*http.Response, error) {
	var response *http.Response
	err := t.limiter.wait(ctx, "get_tweet")
	if err == nil {
		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, "GET", URL+"statuses/show.json", nil)
		if err == nil {
			query := req.URL.Query()
			query.Add("id", tweetID)
			query.Add("include_entities", "true")
			query.Add("include_ext_alt_text", "true")
			query.Add("tweet_mode", "extended")
			req.URL.RawQuery = query.Encode()
			logrus.Debug(fmt.Sprintf("Request URL %s\n", req.URL.String()))
			response, err = t.client.Do(req)
			t.limiter.setLimit("get_tweet", response)
		}
	}
	return response, err
}

func (t *twitter) TweetReply(ctx context.Context, tweetID string, message string) (*Tweet, structured_error.StructuredError) {
	tweet := Tweet{}
	values := url.Values{
		"status":                       []string{message},
		"in_reply_to_status_id":        []string{tweetID},
		"auto_populate_reply_metadata": []string{"true"},
		"include_entities":             []string{"true"},
		"include_ext_alt_text":         []string{"true"},
		"tweet_mode":                   []string{"extended"},
	}
	response, err := t.post(ctx, "tweet_reply", URL+"statuses/update.json", values)
	if err == nil {
		err = GetJSON(response, &tweet)
	}
	return &tweet, structured_error.Wrap(err, structured_error.TwitterError)
}

func (t *twitter) get(ctx context.Context, endpoint string, url string) (*http.Response, error) {
	var request *http.Request
	var response *http.Response
	err := t.limiter.wait(ctx, endpoint)
	if err == nil {
		request, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err == nil {
			response, err = t.client.Do(request)
			t.limiter.setLimit(endpoint, response)
		}
	}
	return response, err
}

func (t *twitter) post(ctx context.Context, endpoint string, url string, data url.Values) (*http.Response, error) {
	var request *http.Request
	var response *http.Response
	err := t.limiter.wait(ctx, endpoint)
	if err == nil {
		request, err = http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(data.Encode()))
		if err == nil {
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response, err = t.client.Do(request)
			t.limiter.setLimit(endpoint, response)
		}
	}
	return response, err
}

func validateStatusCode(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("bad http status code %d", response.StatusCode)
	}
	return nil
}

func GetJSON(response *http.Response, dest interface{}) error {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	logrus.Debug(fmt.Sprintf("Twitter response:\n%v\n", string(body)))

	err = validateStatusCode(response)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, dest)
	return err
}

func getRateLimit(response *http.Response) RateLimit {
	rateLimit := RateLimit{}
	if ceiling, err := strconv.Atoi(response.Header.Get("x-rate-limit-limit")); err == nil {
		rateLimit.Ceiling = &ceiling
	}

	if remaining, err := strconv.Atoi(response.Header.Get("x-rate-limit-remaining")); err == nil {
		rateLimit.Remaining = &remaining
	}

	if nextWindowEpoch, err := strconv.Atoi(response.Header.Get("x-rate-limit-reset")); err == nil {
		nextWindow := time.Unix(int64(nextWindowEpoch), 0)
		rateLimit.NextWindow = &nextWindow
	}
	return rateLimit
}
