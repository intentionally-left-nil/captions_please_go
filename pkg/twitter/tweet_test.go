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
			json:     "{\"id_str\": \"123\", \"text\": \"!hello world!\", \"display_text_range\": [1,12], \"in_reply_to_status_id_str\":\"234\"}",
			expected: Tweet{Id: "123", FullText: "!hello world!", VisibleText: "hello world", ParentTweetId: "234", VisibleTextOffset: 1},
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

func TestTweetTextInfo(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		offset   int
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
			offset:   7,
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
			offset:   7,
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
			offset:   7,
		},
		{
			name:     "Gracefully handles a missing display_text_range for the fallback text",
			json:     "{\"text\": \"hello world\"}",
			expected: "hello world",
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
			name:     "Strips the url from a quote tweet",
			json:     "{\"full_text\": \"https://t.co/foo\", \"display_text_range\":[0,16], \"is_quote_status\": true, \"entities\":{\"urls\":[{\"indices\":[0,16]}]}}",
			expected: "",
		},
		{
			name:     "Strips the url from a quote tweet with text",
			json:     "{\"full_text\": \"hello world https://t.co/foo\", \"display_text_range\":[0,28], \"is_quote_status\": true, \"entities\":{\"urls\":[{\"indices\":[12,28]}]}}",
			expected: "hello world ",
		},
		{
			name:     "Strips the last url from a quote tweet with multiple urls",
			json:     "{\"full_text\": \"hello world https://t.co/foo https://t.co/qt\", \"display_text_range\":[0,44], \"is_quote_status\": true, \"entities\":{\"urls\":[{\"indices\":[12,28]},{\"indices\":[29,44]}]}}",
			expected: "hello world https://t.co/foo ",
		},
		{
			name:     "Ignores the URL from a quote tweet if the indices are invalid",
			json:     "{\"full_text\": \"@captions_please https://t.co/foo\", \"display_text_range\":[17,33], \"is_quote_status\": true, \"entities\":{\"urls\":[{\"indices\":[10,40]}]}}",
			expected: "https://t.co/foo",
			offset:   17,
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
			ti, err := tweet.TextInfo()
			if test.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, ti.Visible())
				assert.Equal(t, test.offset, ti.start)
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
		{
			name:     "retweet of a quote tweet is a retweet",
			json:     `{"is_quote_status": true, "retweeted_status": {}}`,
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
			expected: []Mention{{User: User{Id: "123", Username: "captions_please", Display: "myName"}, StartIndex: 0, EndIndex: 16, Visible: true}},
		},
		{
			name:     "Gracefully handles no entities",
			json:     "{}",
			expected: nil,
		},
		{
			name:     "Message without any visible mentions",
			json:     "{\"text\":\"@captions_please help\", \"display_text_range\":[17, 21], \"entities\":{\"user_mentions\":[{\"id_str\": \"123\", \"screen_name\": \"captions_please\", \"name\": \"myName\", \"indices\":[0,16]}]}}",
			expected: []Mention{{User: User{Id: "123", Username: "captions_please", Display: "myName"}, Visible: false, StartIndex: 0, EndIndex: 16}},
		},
		{
			name: "Message with both visible and invisible mentions",
			json: "{\"text\":\"@captions_please @captions_please help\", \"display_text_range\":[17, 38], \"entities\":{\"user_mentions\":[{\"id_str\": \"123\", \"screen_name\": \"captions_please\", \"name\": \"myName\", \"indices\":[0,16]},{\"id_str\": \"123\", \"screen_name\": \"captions_please\", \"name\": \"myName\", \"indices\":[17,33]}]}}",
			expected: []Mention{
				{User: User{Id: "123", Username: "captions_please", Display: "myName"}, StartIndex: 0, EndIndex: 16, Visible: false},
				{User: User{Id: "123", Username: "captions_please", Display: "myName"}, StartIndex: 17, EndIndex: 33, Visible: true},
			},
		},
		{
			name: "Regression test: very long messages with mentions",
			json: `{
  "display_text_range": [
    14,
    140
  ],
  "entities": {
    "hashtags": [],
    "symbols": [],
    "urls": [
      {
        "display_url": "twitter.com/i/web/status/1…",
        "expanded_url": "https://twitter.com/i/web/status/1436635546097491969",
        "indices": [
          117,
          140
        ],
        "url": "https://t.co/WdenVehFjp"
      }
    ],
    "user_mentions": [
      {
        "id": 464970303,
        "id_str": "464970303",
        "indices": [
          0,
          13
        ],
        "name": "Ronen Steinke",
        "screen_name": "RonenSteinke"
      }
    ]
  },
  "extended_tweet": {
    "display_text_range": [
      14,
      153
    ],
    "entities": {
      "hashtags": [],
      "symbols": [],
      "urls": [],
      "user_mentions": [
        {
          "id": 464970303,
          "id_str": "464970303",
          "indices": [
            0,
            13
          ],
          "name": "Ronen Steinke",
          "screen_name": "RonenSteinke"
        },
        {
          "id": 968802481856794600,
          "id_str": "968802481856794624",
          "indices": [
            124,
            136
          ],
          "name": "Alt Text Reader",
          "screen_name": "get_altText"
        },
        {
          "id": 1264369368386826200,
          "id_str": "1264369368386826240",
          "indices": [
            137,
            153
          ],
          "name": "captions_please",
          "screen_name": "captions_please"
        }
      ]
    },
    "full_text": "@RonenSteinke (weil kein !B im Tweet ist, kurzer Check und ggf auch schon gleich ein automatisiertes Auslesen des Textes.)\n\n@get_altText @captions_please"
  },
  "id": 1436635546097492000,
  "id_str": "1436635546097491969",
  "in_reply_to_screen_name": "RonenSteinke",
  "in_reply_to_status_id": 1436588738973507600,
  "in_reply_to_status_id_str": "1436588738973507584",
  "in_reply_to_user_id": 464970303,
  "in_reply_to_user_id_str": "464970303",
  "is_quote_status": false,
  "text": "@RonenSteinke (weil kein !B im Tweet ist, kurzer Check und ggf auch schon gleich ein automatisiertes Auslesen des T… https://t.co/WdenVehFjp",
  "truncated": true,
  "user": {
    "id_str": "342900911",
    "name": "Freddy",
    "screen_name": "EinfachFreddy"
  }
}`,
			expected: []Mention{
				{User: User{Id: "464970303", Username: "RonenSteinke", Display: "Ronen Steinke"}, StartIndex: 0, EndIndex: 13, Visible: false},
				{User: User{Id: "968802481856794624", Username: "get_altText", Display: "Alt Text Reader"}, StartIndex: 124, EndIndex: 136, Visible: true},
				{User: User{Id: "1264369368386826240", Username: "captions_please", Display: "captions_please"}, StartIndex: 137, EndIndex: 153, Visible: true}},
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
		expected []Media
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
			expected: []Media{{Type: "photo", Url: "https://terminal.space"}},
		},
		{
			name:     "Has media with alt text",
			json:     "{\"extended_entities\": {\"media\":[{\"type\":\"photo\", \"media_url_https\":\"https://terminal.space\", \"ext_alt_text\": \"user caption\"}]}}",
			expected: []Media{{Type: "photo", Url: "https://terminal.space", AltText: &altText}},
		},
		{
			name:     "Has media in the extended tweet",
			json:     "{\"extended_tweet\": {\"extended_entities\": {\"media\":[{\"type\": \"photo\", \"media_url_https\": \"https://terminal.space\"}]}}}",
			expected: []Media{{Type: "photo", Url: "https://terminal.space"}},
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

func TestTweetFallbackMedia(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected []Media
	}{
		{
			name:     "No entities",
			json:     "{}",
			expected: nil,
		},
		{
			name:     "No media",
			json:     "{\"entities\": {}}",
			expected: nil,
		},
		{
			name:     "Empty media",
			json:     "{\"entities\": {\"media\":[]}}",
			expected: nil,
		},
		{
			name:     "Has media",
			json:     "{\"entities\": {\"media\":[{\"type\":\"photo\", \"media_url_https\":\"https://terminal.space\"}]}}",
			expected: []Media{{Type: "photo", Url: "https://terminal.space"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tweet := rawTweet{}
			assert.NoError(t, json.Unmarshal([]byte(test.json), &tweet))
			media := tweet.FallbackMedia()
			assert.Equal(t, test.expected, media)
		})
	}
}
