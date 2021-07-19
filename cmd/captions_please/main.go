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
		api.WriteResponse(w, api.EncodeCRCToken(ctx, req))
	}

	http.HandleFunc("/", handler)
	log.Printf("captions-please listening at http://localhost:%d\n", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil))
}
