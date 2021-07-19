package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
)

func EncodeCRCToken(ctx context.Context, req *http.Request) APIResponse {
	query := req.URL.Query()
	crc_tokens, ok := query["crc_token"]
	if !ok {
		return APIResponse{status: http.StatusBadRequest, response: map[string]string{
			"message": "Error: crc_token missing from request",
		}}
	}

	if len(crc_tokens) != 1 {
		return APIResponse{status: http.StatusBadRequest, response: map[string]string{
			"message": "Error: multiple crc_token query params sent",
		}}
	}

	crc_token := crc_tokens[0]
	h := hmac.New(sha256.New, []byte(GetSecrets(ctx).ConsumerSecret))
	h.Write([]byte(crc_token))
	digest := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return APIResponse{status: http.StatusOK, response: map[string]string{
		"response_token": fmt.Sprintf("sha256=%s", digest),
	}}
}
