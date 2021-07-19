package api

import (
	"context"
	"fmt"
	"os"
	"reflect"
)

type Secrets struct {
	ConsumerSecret string
}

type key int

const theKey key = 0

// unit test indirections
var lookupEnv = os.LookupEnv

func WithSecrets(ctx context.Context) (context.Context, error) {
	data := []struct {
		name string
		env  string
	}{
		{"ConsumerSecret", "TWITTER_CONSUMER_SECRET"},
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
	return withSecrets(ctx, &secrets), nil
}

func GetSecrets(ctx context.Context) *Secrets {
	return ctx.Value(theKey).(*Secrets)
}

func withSecrets(ctx context.Context, secrets *Secrets) context.Context {
	return context.WithValue(ctx, theKey, secrets)
}
