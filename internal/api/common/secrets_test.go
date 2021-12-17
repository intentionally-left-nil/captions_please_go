package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithSecrects(t *testing.T) {
	tests := []struct {
		name            string
		environ         map[string]string
		skipBaseEnviron bool
		hasError        bool
	}{
		{
			name:            "Fails if the key is missing",
			environ:         map[string]string{},
			skipBaseEnviron: true,
			hasError:        true,
		},
		{
			name: "returns the default secrets",
		},
	}

	baseEnviron := map[string]string{
		"TWITTER_CONSUMER_KEY":         "myConsumerKey",
		"TWITTER_CONSUMER_SECRET":      "myConsumerSecret",
		"TWITTER_ACCESS_TOKEN":         "myAccessToken",
		"TWITTER_ACCESS_TOKEN_SECRET":  "myAccessTokenSecret",
		"TWITTER_BEARER_TOKEN":         "myTwitterBearerToken",
		"CAPTIONS_PLEASE_CALLBACK_URL": "myCallbackURL",
		"GOOGLE_PRIVATE_KEY_ID":        "googleyID",
		"GOOGLE_PRIVATE_KEY_SECRET":    "googleySecret",
		"AZURE_COMPUTER_VISION_KEY":    "softieSecret",
		"ASSEMBLY_AI_KEY":              "assemblySecret",
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var environ map[string]string
			if test.skipBaseEnviron {
				environ = assign(map[string]string{}, test.environ)
			} else {
				environ = assign(map[string]string{}, test.environ, baseEnviron)
			}

			mockLookup := func(key string) (string, bool) {
				val, ok := environ[key]
				return val, ok
			}
			originalEnv := lookupEnv
			lookupEnv = mockLookup
			defer func() {
				lookupEnv = originalEnv
			}()

			ctx, err := WithSecrets(context.Background())

			if test.hasError {
				assert.Error(t, err)
				assert.Nil(t, ctx)
			} else {
				assert.Nil(t, err)
				secrets := GetSecrets(ctx)
				assert.NotNil(t, secrets)
				assert.Equal(t, "myConsumerSecret", secrets.TwitterConsumerSecret)
			}
		})
	}
}

func assign(target map[string]string, sources ...map[string]string) map[string]string {
	// Follows the same behavior as Object.assign() in javascript
	numSources := len(sources)
	for i := range sources {
		for key, value := range sources[numSources-1-i] {
			target[key] = value
		}
	}
	return target
}
