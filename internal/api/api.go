package api

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Status   int
	Response interface{}
}

func WriteResponse(w http.ResponseWriter, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.Status)
	err := json.NewEncoder(w).Encode(response.Response)
	if err != nil {
		panic(err)
	}
}
