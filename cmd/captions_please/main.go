package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/AnilRedshift/captions_please_go/internal/api"
	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var PORT = 8080

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "verbose"},
		},
		Before: func(c *cli.Context) error {
			if c.Bool("verbose") {
				logrus.SetLevel(logrus.DebugLevel)
			}
			return nil
		},
	}
	app.Run(os.Args)
	ctx, err := common.WithSecrets(context.Background())
	if err != nil {
		panic(err)
	}

	secrets := common.GetSecrets(ctx)
	client := twitter.NewTwitter(
		secrets.TwitterConsumerKey,
		secrets.TwitterConsumerSecret,
		secrets.TwitterAccessToken,
		secrets.TwitterAccessTokenSecret,
		secrets.TwitterBearerToken)

	config := api.ActivityConfig{
		Workers:            10,
		MaxOutstandingJobs: 9001,
		WebhookTimeout:     time.Second * 30,
	}

	ctx, err = api.WithAccountActivity(ctx, config, client)
	if err != nil {
		panic(err)
	}

	handler := func(w http.ResponseWriter, req *http.Request) {
		var response api.APIResponse
		switch req.Method {
		case http.MethodGet:
			response = api.EncodeCRCToken(ctx, req)
		case http.MethodPost:
			var out <-chan common.ActivityResult
			response, out = api.AccountActivityWebhook(ctx, req)
			go func() {
				for result := range out {
					if result.Err != nil {
						logrus.Error(fmt.Sprintf("%s returned error %v", result.Tweet.Id, result.Err))
					} else {
						tweetId := ""
						if result.Tweet != nil {
							// It's possible the tweet isn't set if we bailed out early in processing
							tweetId = result.Tweet.Id
						}
						logrus.Info(fmt.Sprintf("%s was successfully processed with action %s", tweetId, result.Action))
					}
				}
			}()
		}
		api.WriteResponse(w, response)
	}

	http.HandleFunc("/", handler)
	log.Printf("captions-please listening at http://localhost:%d\n", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil))
}
