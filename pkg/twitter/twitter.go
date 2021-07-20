package twitter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mrjones/oauth"
	log "github.com/sirupsen/logrus"
)

type twitter struct {
	client *http.Client
}

type Twitter interface {
	Webhook() (Webhook, error)
}

type Webhook struct {
	Id    string `json:"id"`
	Url   string `json:"url"`
	Valid bool   `json:"valid"`
}

func NewTwitter(consumerKey string, consumerSecret string, accessToken string, accessTokenSecret string) Twitter {
	c := oauth.NewConsumer(consumerKey, consumerSecret, oauth.ServiceProvider{})
	token := oauth.AccessToken{Token: accessToken, Secret: accessTokenSecret}
	client, _ := c.MakeHttpClient(&token)
	return &twitter{client}
}

func (t *twitter) Webhook() (Webhook, error) {
	var webhook Webhook
	response, err := t.get("account_activity/all/dev/webhooks.json")
	if err == nil {
		statuses := make([]Webhook, 0)
		err = getJSON(response, &statuses)
		if err == nil && len(statuses) == 1 {
			webhook = statuses[0]
		}
	}
	return webhook, err
}

func (t *twitter) get(endpoint string) (*http.Response, error) {
	url := "https://api.twitter.com/1.1/" + endpoint
	return t.client.Get(url)
}

func getJSON(response *http.Response, dest interface{}) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("bad http status code %d", response.StatusCode)
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	log.Debug("Twitter response:\n%v\n", string(body))
	err = json.Unmarshal(body, dest)
	return err
}
