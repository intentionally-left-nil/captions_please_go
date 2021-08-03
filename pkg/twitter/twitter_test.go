package twitter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/stretchr/testify/assert"
)

func TestTwitterLimiterGetSet(t *testing.T) {
	ten := 10
	zero := 0
	future := time.Now().Add(time.Second * 30)
	tests := []struct {
		name         string
		initialLimit *RateLimit
		limit        *RateLimit
		expected     RateLimit
	}{
		{
			name:     "Does nothing if the response is nil",
			expected: RateLimit{},
		},
		{
			name:         "returns the initial rate limit if theres no response",
			initialLimit: &RateLimit{Remaining: &ten, Ceiling: &ten},
			expected:     RateLimit{Remaining: &ten, Ceiling: &ten},
		},
		{
			name:     "Sets the limit to nils if the response doesnt contain anything",
			limit:    &RateLimit{},
			expected: RateLimit{},
		},
		{
			name:     "Ignores setting a new rate limit when not already limited",
			limit:    &RateLimit{Remaining: &ten},
			expected: RateLimit{},
		},
		{
			name:         "Overwrites an expired rate limit with a still expired one",
			initialLimit: &RateLimit{Remaining: &zero},
			limit:        &RateLimit{Remaining: &ten, NextWindow: &future, Ceiling: &ten},
			expected:     RateLimit{Remaining: &ten, NextWindow: &future, Ceiling: &ten},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			limiter := twitterLimiter{}
			if test.initialLimit != nil {
				limiter.limits = map[string]RateLimit{"my_route": *test.initialLimit}
			}

			var response *http.Response
			if test.limit != nil {
				response = &http.Response{Header: http.Header{}}
				if test.limit.Ceiling != nil {
					response.Header.Set("x-rate-limit-limit", fmt.Sprintf("%d", *test.limit.Ceiling))
				}
				if test.limit.Remaining != nil {
					response.Header.Set("x-rate-limit-remaining", fmt.Sprintf("%d", *test.limit.Remaining))
				}
				if test.limit.NextWindow != nil {
					response.Header.Set("x-rate-limit-reset", fmt.Sprintf("%d", test.limit.NextWindow.Unix()))
				}
			}

			limiter.setLimit("my_route", response)

			newLimit := limiter.getLimit("my_route")
			if test.expected.NextWindow == nil {
				assert.Equal(t, test.expected, newLimit)
			} else {
				assert.NotNil(t, newLimit.NextWindow)
				assert.Equal(t, test.expected.NextWindow.Unix(), newLimit.NextWindow.Unix())
				expectedWithoutWindow := test.expected
				expectedWithoutWindow.NextWindow = nil
				actualWithoutWindow := newLimit
				actualWithoutWindow.NextWindow = nil
				assert.Equal(t, expectedWithoutWindow, actualWithoutWindow)
			}
		})
	}
}

func TestTwitterLimiterWait(t *testing.T) {
	stillWaitingErr := errors.New("still waiting yo")
	ctxTimeoutErr := errors.New("The context gave up")
	ten := 10
	zero := 0
	tests := []struct {
		name           string
		contextTimeout time.Duration
		initialLimit   *RateLimit
		windowDuration time.Duration
		expectedErr    error
		expectedWait   time.Duration
	}{
		{
			name:         "Returns immediately if there is no limit",
			expectedErr:  nil,
			expectedWait: 0,
		},
		{
			name:         "Returns immediately if the limit is currently valid",
			initialLimit: &RateLimit{Remaining: &ten},
			expectedErr:  nil,
			expectedWait: 0,
		},
		{
			name:           "Waits for the next window until trying again",
			initialLimit:   &RateLimit{Remaining: &zero},
			windowDuration: time.Millisecond * 50,
			expectedErr:    nil,
			expectedWait:   time.Millisecond * 60,
		},
		{
			name:         "Waits for 30 seconds if no window is given",
			initialLimit: &RateLimit{Remaining: &zero},
			expectedErr:  stillWaitingErr,
			expectedWait: time.Millisecond * 60, // Not actually going to wait the 30 seconds
		},
		{
			name:           "Times out if the context cancels",
			initialLimit:   &RateLimit{Remaining: &zero},
			windowDuration: time.Millisecond * 50,
			contextTimeout: time.Millisecond * 1,
			expectedErr:    ctxTimeoutErr,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			limiter := twitterLimiter{}
			if test.initialLimit != nil {
				if test.windowDuration != 0 {
					future := time.Now().Add(test.windowDuration)
					test.initialLimit.NextWindow = &future
				}
				limiter.limits = map[string]RateLimit{"my_route": *test.initialLimit}
			}

			ctx := context.Background()
			var onComplete context.CancelFunc
			if test.contextTimeout != 0 {
				ctx, onComplete = context.WithTimeout(context.Background(), test.contextTimeout)
			}
			errChan := make(chan error, 1)
			go func() {
				errChan <- limiter.wait(ctx, "my_route")
				if onComplete != nil {
					onComplete()
				}
				close(errChan)
			}()

			upperLimit := test.expectedWait * 2
			if upperLimit == 0 {
				upperLimit = time.Millisecond * 5
			}
			var lowerLimit time.Duration = 0
			if test.expectedWait > 0 {
				lowerLimit = test.expectedWait / 2
			}

			begin := time.Now()
			var err error
			select {
			case err = <-errChan:
			case <-time.After(upperLimit):
				err = stillWaitingErr
			}
			end := time.Now()
			duration := end.Sub(begin)
			assert.GreaterOrEqual(t, duration, lowerLimit)

			switch test.expectedErr {
			case nil:
				assert.Nil(t, err)
			case ctxTimeoutErr:
				assert.Error(t, err)
			case stillWaitingErr:
				assert.Equal(t, stillWaitingErr, err)
			default:
				assert.Fail(t, "unexpected error")
			}
		})
	}
}

func TestValidateResponse(t *testing.T) {
	anError := errors.New("oops")
	tests := []struct {
		name       string
		json       string
		statusCode int
		expected   structured_error.StructuredError
	}{
		{
			name:       "Returns a nil error if the status is 200",
			statusCode: 200,
			expected:   nil,
		},
		{
			name:       "Parses a duplicate error",
			json:       "{\"errors\":[{\"code\":187,\"message\":\"Status is a duplicate.\"}]}",
			statusCode: 400,
			expected:   structured_error.Wrap(anError, structured_error.DuplicateTweetError),
		},
		{
			name:       "Parses a rate limit error",
			json:       "{\"errors\":[{\"code\":88,\"message\":\"staaaahp\"}]}",
			statusCode: 429,
			expected:   structured_error.Wrap(anError, structured_error.RateLimited),
		},
		{
			name:       "Returns a generic Twitter error if unknow",
			json:       "{\"errors\":[{\"code\":999,\"message\":\"staaaahp\"}]}",
			statusCode: 429,
			expected:   structured_error.Wrap(anError, structured_error.TwitterError),
		},
		{
			name:       "Prefers the last error it finds",
			json:       "{\"errors\":[{\"code\":88,\"message\":\"staaaahp\"},{\"code\":187,\"message\":\"Status is a duplicate.\"}]}",
			statusCode: 429,
			expected:   structured_error.Wrap(anError, structured_error.DuplicateTweetError),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := validateResponse(test.statusCode, []byte(test.json))
			if test.expected == nil {
				assert.NoError(t, response)
			} else {
				assert.Error(t, response)
				assert.Equal(t, test.expected.Type(), response.Type())
			}
		})
	}
}
