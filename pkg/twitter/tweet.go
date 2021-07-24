package twitter

import "encoding/json"

type Tweet struct {
	Id            string
	Text          string
	ParentTweetId string
	Type          TweetType
	Mentions      []User
	User          User
	Media         []media
}

type User struct {
	Id       string `json:"id_str"`
	Username string `json:"screen_name"`
	Display  string `json:"name"`
}

type TweetType int

const (
	SimpleTweet TweetType = iota
	QuoteTweet
	Retweet
)

type rawTweet struct {
	Id               string                  `json:"id_str"`
	FullText         string                  `json:"full_text"`
	VisibleRange     []int                   `json:"display_text_range"`
	ExtendedTweet    *extendedTweet          `json:"extended_tweet"`
	Truncated        bool                    `json:"truncated"`
	ParentTweetId    string                  `json:"in_reply_to_user_id_str"`
	QuoteTweet       bool                    `json:"is_quote_status"`
	RetweetedStatus  *map[string]interface{} `json:"retweeted_status"`
	Entities         *entities               `json:"entities"`
	ExtendedEntities *entities               `json:"extended_entities"`
	User             User                    `json:"user"`
}

type extendedTweet struct {
	Text         string `json:"full_text"`
	VisibleRange []int  `json:"display_text_range"`
}

type media struct {
	Id      string  `json:"id_str"`
	Url     string  `json:"media_url_https"`
	Type    string  `json:"type"`
	AltText *string `json:"ext_alt_text"`
}

type entities struct {
	Mentions []User  `json:"user_mentions"`
	Media    []media `json:"media"`
}

func (t *rawTweet) Text() string {
	var (
		text  string
		start int
		end   int
	)
	if t.Truncated &&
		t.ExtendedTweet != nil && t.ExtendedTweet.Text != "" &&
		len(t.ExtendedTweet.VisibleRange) == 2 {
		text = t.ExtendedTweet.Text
		start = t.ExtendedTweet.VisibleRange[0]
		end = t.ExtendedTweet.VisibleRange[1]
	} else if len(t.VisibleRange) == 2 {
		text = t.FullText
		start = t.VisibleRange[0]
		end = t.VisibleRange[1]
	}

	if start < 0 {
		start = 0
	}

	if end >= len(text) {
		end = len(text)
	}

	return text[start:end]
}

func (t *Tweet) UnmarshalJSON(bytes []byte) error {
	raw := rawTweet{}
	err := json.Unmarshal(bytes, &raw)
	if err == nil {
		t.Id = raw.Id
		t.Text = raw.Text()
		t.ParentTweetId = raw.ParentTweetId
		t.User = raw.User

		if raw.QuoteTweet {
			t.Type = QuoteTweet
		} else if raw.RetweetedStatus != nil {
			t.Type = Retweet
		} else {
			t.Type = SimpleTweet
		}

		if raw.Entities != nil && raw.Entities.Mentions != nil {
			t.Mentions = raw.Entities.Mentions
		} else {
			t.Mentions = []User{}
		}

		if raw.ExtendedEntities != nil && raw.ExtendedEntities.Media != nil {
			t.Media = raw.ExtendedEntities.Media
		} else {
			t.Media = []media{}
		}
	}
	return err
}
