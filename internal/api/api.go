package api

import (
	"encoding/json"
	"log"
	"net/http"
)

type APIResponse struct {
	status   int
	response interface{}
}

func WriteResponse(w http.ResponseWriter, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.status)
	err := json.NewEncoder(w).Encode(response.response)
	if err != nil {
		log.Fatal(err)
	}
}
