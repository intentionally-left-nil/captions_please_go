package main

import (
	"context"
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
		Name: "webhook",
		Commands: []*cli.Command{
			{
				Name:   "status",
				Usage:  "Gets a list of all current webhooks",
				Action: status,
			},
			{
				Name:   "create",
				Usage:  "Create a new webhook, pointing to the given URL",
				Action: create,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "url", Required: true},
				},
			},
			{
				Name:   "delete",
				Usage:  "Delete a webhook, given its id",
				Action: delete,
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

func status(c *cli.Context) error {
	client := getClient()
	webhooks, webhookRateLimit, err := client.GetWebhooks(context.Background())
	if err != nil {
		return err
	}

	subscriptions, subscriptionsRateLimit, err := client.GetSubscriptions(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Webhooks")
	printJSON(webhooks)
	printJSON(webhookRateLimit)
	fmt.Println("\nSubscriptions")
	printJSON(subscriptions)
	printJSON(subscriptionsRateLimit)

	return nil
}

func create(c *cli.Context) error {
	client := getClient()
	url := c.String("url")
	webhook, rateLimit, err := client.CreateWebhook(context.Background(), url)
	if err != nil {
		return err
	}
	printJSON(webhook)
	printJSON(rateLimit)
	return nil
}

func delete(c *cli.Context) error {
	client := getClient()
	id := c.String("id")
	rateLimit, err := client.DeleteWebhook(context.Background(), id)
	if err != nil {
		return err
	}
	fmt.Printf("Webhook %s successfully deleted\n", id)
	printJSON(rateLimit)
	return nil
}

func onBefore(c *cli.Context) error {
	if c.Bool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	return nil
}

func printJSON(v interface{}) {
	message, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(message))
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
