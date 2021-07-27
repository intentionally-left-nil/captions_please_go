package twitter

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalTweet(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected Tweet
		hasError bool
	}{
		{
			name:     "Parses a valid tweet",
			json:     "{\"id_str\": \"123\", \"text\": \"!hello world!\", \"display_text_range\": [1,12]}",
			expected: Tweet{Id: "123", Text: "hello world"},
		},
		{
			name:     "Errors if the id is missing",
			json:     "{\"text\": \"!hello world!\", \"display_text_range\": [1,12]}",
			hasError: true,
		},
		{
			name:     "Errors if the text is invalid",
			json:     "{\"id_str\": \"123\", \"text\": \"!hello world!\", \"display_text_range\": [-1,12]}",
			hasError: true,
		},
		{
			name:     "Errors if the mentions are invalid",
			json:     "{\"id_str\": \"123\", \"text\": \"!hello world!\", \"display_text_range\": [1,12], \"entities\":{\"user_mentions\":[{}]}}",
			hasError: true,
		},
		{
			name:     "Errors if the json is invalid",
			json:     "{\"id_str\":123}",
			hasError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tweet := Tweet{}
			err := json.Unmarshal([]byte(test.json), &tweet)
			if test.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, tweet)
			}
		})
	}
}

func TestTweetText(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string
		hasError bool
	}{
		{
			name:     "Returns the extended text",
			json:     "{\"truncated\": true, \"extended_tweet\":{\"full_text\":\"hello world\", \"display_text_range\":[0,11]}}",
			expected: "hello world",
		},
		{
			name:     "Returns a slice of the extended text",
			json:     "{\"truncated\": true, \"extended_tweet\":{\"full_text\":\"hello world\", \"display_text_range\":[7,10]}}",
			expected: "orl",
		},
		{
			name:     "Returns the full text of a non-truncated tweet",
			json:     "{\"full_text\": \"hello world\", \"display_text_range\":[0,11]}",
			expected: "hello world",
		},
		{
			name:     "Returns a slice of a non-truncated tweet",
			json:     "{\"full_text\": \"hello world\", \"display_text_range\":[7,10]}",
			expected: "orl",
		},
		{
			name:     "Returns the fallback text of a non-truncated tweet",
			json:     "{\"text\": \"hello world\", \"display_text_range\":[0,11]}",
			expected: "hello world",
		},
		{
			name:     "Returns a fallback slice of a truncated tweet",
			json:     "{\"text\": \"hello world\", \"display_text_range\":[7,10]}",
			expected: "orl",
		},
		{
			name:     "Prefers the extended text",
			json:     "{\"truncated\": true, \"full_text\": \"chose the wrong text\", \"text\": \"also chose the wrong text\", \"extended_tweet\":{\"full_text\":\"hello world\", \"display_text_range\":[0,11]}}",
			expected: "hello world",
		},
		{
			name:     "Prefers the full_text if not truncated",
			json:     "{\"truncated\": false, \"full_text\": \"hello full_text\", \"text\": \"also chose the wrong text\", \"display_text_range\":[0,15], \"extended_tweet\":{\"full_text\":\"wrong one\", \"display_text_range\":[0,11]}}",
			expected: "hello full_text",
		},
		{
			name:     "Falls back to the text if full_text is missing",
			json:     "{\"truncated\": false, \"text\": \"hello text\", \"display_text_range\":[0,10], \"extended_tweet\":{\"full_text\":\"wrong text\", \"display_text_range\":[0,10]}}",
			expected: "hello text",
		},
		{
			name:     "Errors with no data",
			json:     "{}",
			hasError: true,
		},
		{
			name:     "errors if truncated but no extended_tweet",
			json:     "{\"truncated\": true, \"text\": \"hello world\", \"display_text_range\":[7,10]}",
			hasError: true,
		},
		{
			name:     "Errors if the extended_tweet full_text is missing",
			json:     "{\"truncated\": true, \"extended_tweet\":{\"display_text_range\":[0,0]}}",
			hasError: true,
		},
		{
			name:     "Errors if the extended_tweet len is invalid",
			json:     "{\"truncated\": true, \"extended_tweet\":{\"full_text\":\"hello world\"}}",
			hasError: true,
		},
		{
			name:     "Errors if the extended_tweet start is negative",
			json:     "{\"truncated\": true, \"extended_tweet\":{\"full_text\":\"hello world\", \"display_text_range\":[-1,11]}}",
			hasError: true,
		},
		{
			name:     "Errors if the extended_tweet start is greater than the end",
			json:     "{\"truncated\": true, \"extended_tweet\":{\"full_text\":\"hello world\", \"display_text_range\":[6,5]}}",
			hasError: true,
		},
		{
			name:     "Errors if the extended_tweet end is greater than the length",
			json:     "{\"truncated\": true, \"extended_tweet\":{\"full_text\":\"hello world\", \"display_text_range\":[0,12]}}",
			hasError: true,
		},
		{
			name:     "Errors if display_text_range is invalid",
			json:     "{\"full_text\": \"hello world\"}",
			hasError: true,
		},
		{
			name:     "Errors if missing both the full_text and the text",
			json:     "{\"display_text_range\":[0,11]}",
			hasError: true,
		},
		{
			name:     "Errors if the start is negative",
			json:     "{\"full_text\": \"hello world\", \"display_text_range\":[-1,11]}",
			hasError: true,
		},
		{
			name:     "Errors if the start is greater than the end",
			json:     "{\"full_text\": \"hello world\", \"display_text_range\":[6,5]}",
			hasError: true,
		},
		{
			name:     "Errors if the end is too long",
			json:     "{\"full_text\": \"hello world\", \"display_text_range\":[0,12]}",
			hasError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tweet := rawTweet{}
			assert.NoError(t, json.Unmarshal([]byte(test.json), &tweet))
			text, err := tweet.Text()
			if test.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, text)
			}
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
			tweet := rawTweet{}
			assert.NoError(t, json.Unmarshal([]byte(test.json), &tweet))
			assert.Equal(t, test.expected, tweet.TweetType())
		})
	}
}

func TestTweetMentions(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected []Mention
		hasError bool
	}{
		{
			name:     "Returns a mention",
			json:     "{\"text\":\"@captions_please help\", \"display_text_range\":[0, 21], \"entities\":{\"user_mentions\":[{\"id_str\": \"123\", \"screen_name\": \"captions_please\", \"name\": \"myName\", \"indices\":[0,16]}]}}",
			expected: []Mention{{User: User{Id: "123", Username: "captions_please", Display: "myName"}, StartIndex: 0, EndIndex: 16}},
		},
		{
			name:     "Gracefully handles no entities",
			json:     "{}",
			expected: nil,
		},
		{
			name:     "Message without any mentions",
			json:     "{\"entities\":{}}",
			expected: nil,
		},
		{
			name:     "Errors if getting the tweet text fails",
			json:     "{\"text\":\"@captions_please help\", \"display_text_range\":[-1, 21], \"entities\":{\"user_mentions\":[{\"id_str\": \"123\", \"screen_name\": \"captions_please\", \"name\": \"myName\", \"indices\":[0,16]}]}}",
			hasError: true,
		},
		{
			name:     "Errors if the range is invalid",
			json:     "{\"text\":\"@captions_please help\", \"display_text_range\":[0, 21], \"entities\":{\"user_mentions\":[{\"id_str\": \"123\", \"screen_name\": \"captions_please\", \"name\": \"myName\", \"indices\":[5,22]}]}}",
			hasError: true,
		},
		{
			name:     "Errors if the id is invalid",
			json:     "{\"text\":\"@captions_please help\", \"display_text_range\":[0, 21], \"entities\":{\"user_mentions\":[{\"screen_name\": \"captions_please\", \"name\": \"myName\", \"indices\":[0,16]}]}}",
			hasError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tweet := rawTweet{}
			assert.NoError(t, json.Unmarshal([]byte(test.json), &tweet))
			mentions, err := tweet.Mentions()
			if test.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, mentions)
			}
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
			expected: nil,
		},
		{
			name:     "No media",
			json:     "{\"extended_entities\": {}}",
			expected: nil,
		},
		{
			name:     "Empty media",
			json:     "{\"extended_entities\": {\"media\":[]}}",
			expected: nil,
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
			tweet := rawTweet{}
			assert.NoError(t, json.Unmarshal([]byte(test.json), &tweet))
			media := tweet.Media()
			assert.Equal(t, test.expected, media)
		})
	}
}
