package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/AnilRedshift/captions_please_go/internal/api"
)

var PORT = 8080

func main() {
	ctx, err := api.WithSecrets(context.Background())
	if err != nil {
		panic(err)
	}

	handler := func(w http.ResponseWriter, req *http.Request) {
		var response api.APIResponse
		switch req.Method {
		case http.MethodGet:
			response = api.EncodeCRCToken(ctx, req)
		case http.MethodPost:
			response = api.AccountActivityWebhook(ctx, req)
		}
		api.WriteResponse(w, response)
	}

	http.HandleFunc("/", handler)
	log.Printf("captions-please listening at http://localhost:%d\n", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil))
}
