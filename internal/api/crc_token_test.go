package api

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeCRCToken(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		status         int
		consumerSecret string
		digest         string
	}{
		{
			name:   "Returns 400 when not passed a crc_token",
			url:    "https://terminal.space",
			status: 400,
		},
		{
			name:   "Returns 400 when passed multiple crc_tokens",
			url:    "https://terminal.space?crc_token=1&crc_token=2",
			status: 400,
		},
		{
			name:           "Encodes a response",
			url:            "https://terminal.space?crc_token=abc",
			status:         200,
			consumerSecret: "abc",
			digest:         "sha256=LwLiSuLh/ogDmfJ2AK+og2TmBiv5u+EUsy+o8j0DYIo=",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url, err := url.Parse(test.url)
			if assert.NoError(t, err) {
				req := &http.Request{URL: url}
				secrets := &Secrets{TwitterConsumerSecret: test.consumerSecret}
				ctx := withSecrets(context.Background(), secrets)
				response := EncodeCRCToken(ctx, req)
				assert.Equal(t, test.status, response.status)

				if test.status == http.StatusOK {
					assert.NotEmpty(t, test.digest, "Test is invalid, digest must be specified")
					token := response.response.(map[string]string)["response_token"]
					assert.Equal(t, test.digest, token)
				}
			}
		})
	}
}
