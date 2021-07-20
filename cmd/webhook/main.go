package main

import (
	"flag"
	"fmt"

	"github.com/AnilRedshift/captions_please_go/internal/api"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	log "github.com/sirupsen/logrus"
)

func main() {
	verbose := flag.Bool("verbose", false, "Turn on verbose logging")
	flag.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}
	if len(flag.Args()) != 1 {
		help()
	}
	switch flag.Arg(0) {
	case "status":
		status()
	default:
		help()
	}
}

func status() {
	client := getClient()
	webhook, err := client.Webhook()
	if err != nil {
		panic(err)
	}
	if webhook.Valid {
		fmt.Printf("Webhook ID: %s points to %s\n", webhook.Id, webhook.Url)
	} else {
		panic(fmt.Errorf("webhook %s is invalid", webhook.Id))
	}
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

func help() {
	fmt.Println("usage: webhook status")
	flag.PrintDefaults()
}
