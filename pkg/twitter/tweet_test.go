package twitter

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTweetText(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string
	}{
		{
			name:     "Returns an empty string with no data",
			json:     "{}",
			expected: "",
		},
		{
			name:     "Returns an empty string without a range",
			json:     "{\"full_text\": \"hello\"}",
			expected: "",
		},
		{
			name:     "Returns the whole full_text",
			json:     "{\"full_text\": \"hello\", \"display_text_range\":[0, 5]}",
			expected: "hello",
		},
		{
			name:     "Returns a slice of the full_text",
			json:     "{\"full_text\": \"hello\", \"display_text_range\":[2, 4]}",
			expected: "ll",
		},
		{
			name:     "Handles a negative start index",
			json:     "{\"full_text\": \"hello\", \"display_text_range\":[-1, 1]}",
			expected: "h",
		},
		{
			name:     "Handles an end index too long",
			json:     "{\"full_text\": \"hello\", \"display_text_range\":[0, 6]}",
			expected: "hello",
		},
		{
			name:     "Returns an empty string if the VisibleRange is invalid",
			json:     "{\"full_text\": \"hello\", \"display_text_range\":[0]}",
			expected: "",
		},
		{
			name:     "Ignores the extended text if not truncated",
			json:     "{\"full_text\": \"hello\", \"display_text_range\":[0, 5], \"truncated\": false, \"extended_tweet\":{\"full_text\":\"hello world\", \"display_text_range\":[0,11]}}",
			expected: "hello",
		},
		{
			name:     "Ignores the extended text if the field is missing",
			json:     "{\"full_text\": \"hello\", \"display_text_range\":[0, 5], \"truncated\": true}",
			expected: "hello",
		},
		{
			name:     "Ignores the extended text if the range is invalid",
			json:     "{\"full_text\": \"hello\", \"display_text_range\":[0, 5], \"truncated\": true, \"extended_tweet\":{\"full_text\":\"hello world\", \"display_text_range\":[]}}",
			expected: "hello",
		},
		{
			name:     "Returns the extended tweet",
			json:     "{\"full_text\": \"hello\", \"display_text_range\":[0, 5], \"truncated\": true, \"extended_tweet\":{\"full_text\":\"hello world\", \"display_text_range\":[0,11]}}",
			expected: "hello world",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tweet := Tweet{}
			assert.NoError(t, json.Unmarshal([]byte(test.json), &tweet))
			assert.Equal(t, test.expected, tweet.Text)
		})
	}
}

func TestTweetType(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected TweetType
	}{
		{
			name:     "Is a quote tweet if true",
			json:     "{\"is_quote_status\": true}",
			expected: QuoteTweet,
		},
		{
			name:     "Ignores quote_status false",
			json:     "{\"is_quote_status\": false}",
			expected: SimpleTweet,
		},
		{
			name:     "Is a retweet",
			json:     "{\"retweeted_status\": {}}",
			expected: Retweet,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tweet := Tweet{}
			assert.NoError(t, json.Unmarshal([]byte(test.json), &tweet))
			assert.Equal(t, test.expected, tweet.Type)
		})
	}
}

func TestTweetMentions(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected []User
	}{
		{
			name:     "No entities",
			json:     "{}",
			expected: []User{},
		},
		{
			name:     "No mentions in the entity",
			json:     "{\"entities\":{}}",
			expected: []User{},
		},
		{
			name:     "Empty user mentions",
			json:     "{\"entities\":{\"user_mentions\":[]}}",
			expected: []User{},
		},
		{
			name:     "Has mentions",
			json:     "{\"entities\":{\"user_mentions\":[{\"id_str\": \"123\", \"screen_name\":\"captions_please\"}, {}]}}",
			expected: []User{{Id: "123", Username: "captions_please"}, {}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tweet := Tweet{}
			assert.NoError(t, json.Unmarshal([]byte(test.json), &tweet))
			assert.Equal(t, test.expected, tweet.Mentions)
		})
	}
}

func TestTweetMedia(t *testing.T) {

	altText := "user caption"
	tests := []struct {
		name     string
		json     string
		expected []media
	}{
		{
			name:     "No extended_entities",
			json:     "{}",
			expected: []media{},
		},
		{
			name:     "No media",
			json:     "{\"extended_entities\": {}}",
			expected: []media{},
		},
		{
			name:     "Empty media",
			json:     "{\"extended_entities\": {\"media\":[]}}",
			expected: []media{},
		},
		{
			name:     "Has media",
			json:     "{\"extended_entities\": {\"media\":[{\"type\":\"photo\", \"media_url_https\":\"https://terminal.space\"}]}}",
			expected: []media{{Type: "photo", Url: "https://terminal.space"}},
		},
		{
			name:     "Has media with alt text",
			json:     "{\"extended_entities\": {\"media\":[{\"type\":\"photo\", \"media_url_https\":\"https://terminal.space\", \"ext_alt_text\": \"user caption\"}]}}",
			expected: []media{{Type: "photo", Url: "https://terminal.space", AltText: &altText}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tweet := Tweet{}
			assert.NoError(t, json.Unmarshal([]byte(test.json), &tweet))
			assert.Equal(t, test.expected, tweet.Media)
		})
	}
}
