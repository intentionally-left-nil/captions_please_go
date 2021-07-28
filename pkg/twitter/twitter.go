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
	"time"

	"github.com/mrjones/oauth"
	"github.com/sirupsen/logrus"
)

type twitter struct {
	client *http.Client
	bearer string
}

type RateLimit struct {
	Ceiling    *int
	Remaining  *int
	NextWindow *time.Time
}

type Twitter interface {
	GetWebhooks(ctx context.Context) ([]Webhook, RateLimit, error)
	CreateWebhook(ctx context.Context, url string) (Webhook, RateLimit, error)
	DeleteWebhook(ctx context.Context, webhookID string) (RateLimit, error)
	GetSubscriptions(ctx context.Context) ([]Subscription, RateLimit, error)
	DeleteSubscription(ctx context.Context, subscriptionID string) (RateLimit, error)
	AddSubscription(ctx context.Context) (RateLimit, error)
	GetTweetRaw(ctx context.Context, tweetID string) (*http.Response, RateLimit, error)
	GetTweet(ctx context.Context, tweetID string) (*Tweet, RateLimit, error)
	TweetReply(ctx context.Context, tweetID string, message string) (*Tweet, RateLimit, error)
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

func (t *twitter) GetWebhooks(ctx context.Context) ([]Webhook, RateLimit, error) {
	var webhooks []Webhook
	response, err := t.get(ctx, URL+"account_activity/all/dev/webhooks.json")
	if err == nil {
		webhooks = make([]Webhook, 0)
		err = GetJSON(response, &webhooks)
	}
	return webhooks, getRateLimit(response), err
}

func (t *twitter) DeleteWebhook(ctx context.Context, webhookID string) (RateLimit, error) {
	url := fmt.Sprintf("%saccount_activity/all/dev/webhooks/%s.json", URL, webhookID)
	request, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	rateLimit := RateLimit{}
	if err == nil {
		var response *http.Response
		response, err = t.client.Do(request)
		rateLimit = getRateLimit(response)
		if err == nil {
			var body []byte
			body, err = ioutil.ReadAll(response.Body)
			logrus.Debug(fmt.Sprintf("Twitter response:\n%v\n", string(body)))
			if err == nil {
				err = validateStatusCode(response)
			}
		}
	}
	return rateLimit, err
}

func (t *twitter) CreateWebhook(ctx context.Context, webhookUrl string) (Webhook, RateLimit, error) {
	var webhook Webhook
	response, err := t.post(ctx, URL+"account_activity/all/dev/webhooks.json", url.Values{"url": []string{webhookUrl}})
	if err == nil {
		err = GetJSON(response, &webhook)
	}
	return webhook, getRateLimit(response), err
}

func (t *twitter) GetSubscriptions(ctx context.Context) ([]Subscription, RateLimit, error) {
	type apiResponse struct {
		Subscriptions []Subscription `json:"subscriptions"`
	}
	var subscriptions []Subscription
	rateLimit := RateLimit{}
	request, err := http.NewRequestWithContext(ctx, "GET", URL+"account_activity/all/dev/subscriptions/list.json", nil)
	if err == nil {
		request.Header.Set("Authorization", "Bearer "+t.bearer)
		client := http.Client{}
		var response *http.Response
		response, err = client.Do(request)
		rateLimit = getRateLimit(response)
		if err == nil {
			api := apiResponse{}
			err = GetJSON(response, &api)
			if err == nil {
				subscriptions = api.Subscriptions
			}
		}
	}
	return subscriptions, rateLimit, err
}

func (t *twitter) DeleteSubscription(ctx context.Context, subscriptionID string) (RateLimit, error) {
	url := fmt.Sprintf("%saccount_activity/all/dev/subscriptions/%s.json", URL, subscriptionID)
	request, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	rateLimit := RateLimit{}
	if err == nil {
		request.Header.Set("Authorization", "Bearer "+t.bearer)
		client := http.Client{}
		var response *http.Response
		response, err = client.Do(request)
		rateLimit = getRateLimit(response)
		if err == nil {
			var body []byte
			body, err = ioutil.ReadAll(response.Body)
			logrus.Debug(fmt.Sprintf("Twitter response:\n%v\n", string(body)))
			if err == nil {
				err = validateStatusCode(response)
			}
		}
	}
	return rateLimit, err
}

func (t *twitter) AddSubscription(ctx context.Context) (RateLimit, error) {
	response, err := t.post(ctx, URL+"account_activity/all/dev/subscriptions.json", url.Values{})
	if err == nil {
		var body []byte
		body, err = ioutil.ReadAll(response.Body)
		logrus.Debug(fmt.Sprintf("Twitter response:\n%v\n", string(body)))
		if err == nil {
			err = validateStatusCode(response)
		}
	}
	return getRateLimit(response), err
}
func (t *twitter) GetTweet(ctx context.Context, tweetID string) (*Tweet, RateLimit, error) {
	tweet := Tweet{}
	response, rateLimit, err := t.GetTweetRaw(ctx, tweetID)
	if err == nil {
		rateLimit = getRateLimit(response)
		if err == nil {
			err = GetJSON(response, &tweet)
		}
	}
	return &tweet, rateLimit, err
}

func (t *twitter) GetTweetRaw(ctx context.Context, tweetID string) (*http.Response, RateLimit, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", URL+"statuses/show.json", nil)
	rateLimit := RateLimit{}
	var response *http.Response
	if err == nil {
		query := req.URL.Query()
		query.Add("id", tweetID)
		query.Add("include_entities", "true")
		query.Add("include_ext_alt_text", "true")
		query.Add("tweet_mode", "extended")
		req.URL.RawQuery = query.Encode()
		logrus.Debug(fmt.Sprintf("Request URL %s\n", req.URL.String()))
		response, err = t.client.Do(req)
		rateLimit = getRateLimit(response)
	}
	return response, rateLimit, err
}

func (t *twitter) TweetReply(ctx context.Context, tweetID string, message string) (*Tweet, RateLimit, error) {
	tweet := Tweet{}
	values := url.Values{
		"status":                       []string{message},
		"in_reply_to_status_id":        []string{tweetID},
		"auto_populate_reply_metadata": []string{"true"},
	}
	response, err := t.post(ctx, URL+"statuses/update.json", values)
	if err == nil {
		err = GetJSON(response, &tweet)
	}
	return &tweet, getRateLimit(response), err
}

func (t *twitter) get(ctx context.Context, url string) (*http.Response, error) {
	var response *http.Response
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err == nil {
		response, err = t.client.Do(request)
	}
	return response, err
}

func (t *twitter) post(ctx context.Context, url string, data url.Values) (*http.Response, error) {
	var response *http.Response
	request, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(data.Encode()))
	if err == nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response, err = t.client.Do(request)
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
