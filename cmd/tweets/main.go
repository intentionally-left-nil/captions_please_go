package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/AnilRedshift/captions_please_go/internal/api"
	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name: "tweets",
		Commands: []*cli.Command{
			{
				Name:   "get",
				Usage:  "Gets a tweet by its id",
				Action: getTweet,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
				},
			},
			{
				Name:   "reply",
				Usage:  "Replies to a tweet with a message",
				Action: tweetReply,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
					&cli.StringFlag{Name: "message", Required: true},
				},
			},
			{
				Name:   "process",
				Usage:  "Respond to a tweet as if it were the webhook",
				Action: processTweet,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "verbose"},
		},
		Before: onBefore,
	}
	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}

func getTweet(c *cli.Context) error {
	client := getClient()
	tweet, err := client.GetTweet(context.Background(), c.String("id"))
	if err == nil {
		printJSON(tweet)

	}
	return err
}

func processTweet(c *cli.Context) error {
	client := getClient()
	var err error
	response, err := client.GetTweetRaw(context.Background(), c.String("id"))
	if err == nil {
		createData := map[string]interface{}{}
		twitter.GetJSON(response, &createData)
		var activityJSON []byte
		activityJSON, err = json.Marshal(map[string]interface{}{
			"tweet_create_events": []interface{}{createData},
			// TODO: don't hardcode the bot ID
			"for_user_id": "1264369368386826240",
		})

		if err == nil {
			fmt.Println("Got the tweet, processing the webhook")
			reader := io.NopCloser(strings.NewReader(string(activityJSON)))
			request := &http.Request{Body: reader}
			var ctx context.Context
			ctx, err = common.WithSecrets(context.Background())
			if err == nil {
				ctx, err = api.WithAccountActivity(ctx, api.ActivityConfig{}, client)
				if err == nil {
					_, out := api.AccountActivityWebhook(ctx, request)
					for result := range out {
						printJSON(result)
					}
				}
			}
		}
	}
	return err
}

func tweetReply(c *cli.Context) error {
	client := getClient()
	tweet, err := client.TweetReply(context.Background(), c.String("id"), c.String("message"))
	if err == nil {
		printJSON(tweet)
	}
	return err
}

func onBefore(c *cli.Context) error {
	if c.Bool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	return nil
}

func getClient() twitter.Twitter {
	secrets, err := common.NewSecrets()
	if err != nil {
		panic(err)
	}
	return twitter.NewTwitter(
		secrets.TwitterConsumerKey,
		secrets.TwitterConsumerSecret,
		secrets.TwitterAccessToken,
		secrets.TwitterAccessTokenSecret,
		secrets.TwitterBearerToken)
}

func printJSON(v interface{}) {
	message, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(message))
}
