package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AnilRedshift/captions_please_go/internal/api"
	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
)

var PORT = 8080

func main() {
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
				for range out {
					// Just drain the result, it's already logged in the pipeline
				}
			}()
		}
		api.WriteResponse(w, response)
	}

	http.HandleFunc("/", handler)
	log.Printf("captions-please listening at http://localhost:%d\n", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil))
}
