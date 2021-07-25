package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AnilRedshift/captions_please_go/internal/api"
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
	tweet, rateLimit, err := client.GetTweet(c.String("id"))
	if err == nil {
		printJSON(tweet)
		printJSON(rateLimit)

	}
	return err
}

func tweetReply(c *cli.Context) error {
	client := getClient()
	tweet, rateLimit, err := client.TweetReply(c.String("id"), c.String("message"))
	if err == nil {
		printJSON(tweet)
		printJSON(rateLimit)
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
	secrets, err := api.NewSecrets()
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
