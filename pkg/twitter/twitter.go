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
}

type Twitter interface {
	GetWebhooks() ([]Webhook, error)
	CreateWebhook(url string) (Webhook, error)
	DeleteWebhook(webhookID string) error
}

type Webhook struct {
	Id    string `json:"id"`
	Url   string `json:"url"`
	Valid bool   `json:"valid"`
}

const URL = "https://api.twitter.com/1.1/"

func NewTwitter(consumerKey string, consumerSecret string, accessToken string, accessTokenSecret string) Twitter {
	c := oauth.NewConsumer(consumerKey, consumerSecret, oauth.ServiceProvider{})
	token := oauth.AccessToken{Token: accessToken, Secret: accessTokenSecret}
	client, _ := c.MakeHttpClient(&token)
	return &twitter{client}
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
			if err != nil {
				return err
			}
			logrus.Debug("Twitter response:\n%v\n", string(body))
			err = validateStatusCode(response)
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
	logrus.Debug("Twitter response:\n%v\n", string(body))

	err = validateStatusCode(response)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, dest)
	return err
}
