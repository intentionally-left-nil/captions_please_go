package twitter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/mrjones/oauth"
	"github.com/sirupsen/logrus"
)

type twitter struct {
	client *http.Client
	bearer string
}

type Twitter interface {
	GetWebhooks() ([]Webhook, error)
	CreateWebhook(url string) (Webhook, error)
	DeleteWebhook(webhookID string) error
	GetSubscriptions() ([]Subscription, error)
	DeleteSubscription(subscriptionID string) error
	AddSubscription() error
	GetTweet(tweetID string) (*Tweet, error)
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

func (t *twitter) GetWebhooks() ([]Webhook, error) {
	var webhooks []Webhook
	response, err := t.client.Get(URL + "account_activity/all/dev/webhooks.json")
	if err == nil {
		webhooks = make([]Webhook, 0)
		err = getJSON(response, &webhooks)
	}
	return webhooks, err
}

func (t *twitter) DeleteWebhook(webhookID string) error {
	url := fmt.Sprintf("%saccount_activity/all/dev/webhooks/%s.json", URL, webhookID)
	request, err := http.NewRequest("DELETE", url, nil)
	if err == nil {
		var response *http.Response
		response, err = t.client.Do(request)
		if err == nil {
			var body []byte
			body, err = ioutil.ReadAll(response.Body)
			logrus.Debug(fmt.Sprintf("Twitter response:\n%v\n", string(body)))
			if err == nil {
				err = validateStatusCode(response)
			}
		}
	}
	return err
}

func (t *twitter) CreateWebhook(webhookUrl string) (Webhook, error) {
	var webhook Webhook
	response, err := t.client.PostForm(URL+"account_activity/all/dev/webhooks.json", url.Values{"url": []string{webhookUrl}})
	if err == nil {
		err = getJSON(response, &webhook)
	}
	return webhook, err
}

func (t *twitter) GetSubscriptions() ([]Subscription, error) {
	type apiResponse struct {
		Subscriptions []Subscription `json:"subscriptions"`
	}
	var subscriptions []Subscription
	request, err := http.NewRequest("GET", URL+"account_activity/all/dev/subscriptions/list.json", nil)
	if err == nil {
		request.Header.Set("Authorization", "Bearer "+t.bearer)
		client := http.Client{}
		var response *http.Response
		response, err = client.Do(request)
		if err == nil {
			api := apiResponse{}
			err = getJSON(response, &api)
			if err == nil {
				subscriptions = api.Subscriptions
			}
		}
	}
	return subscriptions, err
}

func (t *twitter) DeleteSubscription(subscriptionID string) error {
	url := fmt.Sprintf("%saccount_activity/all/dev/subscriptions/%s.json", URL, subscriptionID)
	request, err := http.NewRequest("DELETE", url, nil)
	if err == nil {
		request.Header.Set("Authorization", "Bearer "+t.bearer)
		client := http.Client{}
		var response *http.Response
		response, err = client.Do(request)
		if err == nil {
			var body []byte
			body, err = ioutil.ReadAll(response.Body)
			logrus.Debug(fmt.Sprintf("Twitter response:\n%v\n", string(body)))
			if err == nil {
				err = validateStatusCode(response)
			}
		}
	}
	return err
}

func (t *twitter) AddSubscription() error {
	response, err := t.client.PostForm(URL+"account_activity/all/dev/subscriptions.json", url.Values{})
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

func (t *twitter) GetTweet(tweetID string) (*Tweet, error) {
	req, err := http.NewRequest("GET", URL+"statuses/show.json", nil)
	tweet := Tweet{}
	if err == nil {
		query := req.URL.Query()
		query.Add("id", tweetID)
		query.Add("include_entities", "true")
		query.Add("include_ext_alt_text", "true")
		query.Add("tweet_mode", "extended")
		req.URL.RawQuery = query.Encode()
		logrus.Debug(fmt.Sprintf("Request URL %s\n", req.URL.String()))
		var response *http.Response
		response, err = t.client.Do(req)
		if err == nil {
			err = getJSON(response, &tweet)
		}
	}
	return &tweet, err
}

func validateStatusCode(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("bad http status code %d", response.StatusCode)
	}
	return nil
}

func getJSON(response *http.Response, dest interface{}) error {
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
