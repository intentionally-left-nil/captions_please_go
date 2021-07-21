package api

import (
	"context"
	"fmt"
	"os"
	"reflect"
)

type Secrets struct {
	TwitterConsumerKey       string
	TwitterConsumerSecret    string
	TwitterAccessToken       string
	TwitterAccessTokenSecret string
	TwitterBearerToken       string
	WebhookUrl               string
	GooglePrivateKeyID       string
	GooglePrivateKeySecret   string
}

type key int

const theKey key = 0

// unit test indirections
var lookupEnv = os.LookupEnv

func NewSecrets() (*Secrets, error) {
	data := []struct {
		name string
		env  string
	}{
		{"TwitterConsumerKey", "TWITTER_CONSUMER_KEY"},
		{"TwitterConsumerSecret", "TWITTER_CONSUMER_SECRET"},
		{"TwitterAccessToken", "TWITTER_ACCESS_TOKEN"},
		{"TwitterAccessTokenSecret", "TWITTER_ACCESS_TOKEN_SECRET"},
		{"TwitterBearerToken", "TWITTER_BEARER_TOKEN"},
		{"WebhookUrl", "CAPTIONS_PLEASE_CALLBACK_URL"},
		{"GooglePrivateKeyID", "GOOGLE_PRIVATE_KEY_ID"},
		{"GooglePrivateKeySecret", "GOOGLE_PRIVATE_KEY_SECRET"},
	}

	secrets := Secrets{}
	for _, item := range data {
		field := reflect.ValueOf(&secrets).Elem().FieldByName(item.name)
		secret, ok := lookupEnv(item.env)
		if !ok || secret == "" {
			return nil, fmt.Errorf("missing %s secret", item.env)
		}
		field.Set(reflect.ValueOf(secret))
	}
	return &secrets, nil
}

func WithSecrets(ctx context.Context) (context.Context, error) {
	secrets, err := NewSecrets()
	if err != nil {
		return nil, err
	}
	return withSecrets(ctx, secrets), nil
}

func GetSecrets(ctx context.Context) *Secrets {
	return ctx.Value(theKey).(*Secrets)
}

func withSecrets(ctx context.Context, secrets *Secrets) context.Context {
	return context.WithValue(ctx, theKey, secrets)
}
