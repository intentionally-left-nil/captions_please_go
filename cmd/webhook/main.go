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
	webhooks, err := client.GetWebhooks()
	if err != nil {
		return err
	}
	printJSON(webhooks)
	return nil
}

func create(c *cli.Context) error {
	client := getClient()
	url := c.String("url")
	webhook, err := client.CreateWebhook(url)
	if err != nil {
		return err
	}
	printJSON(webhook)
	return nil
}

func delete(c *cli.Context) error {
	client := getClient()
	id := c.String("id")
	err := client.DeleteWebhook(id)
	if err != nil {
		return err
	}
	fmt.Printf("Webhook %s successfully deleted\n", id)
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
		secrets.TwitterAccessTokenSecret)
}
