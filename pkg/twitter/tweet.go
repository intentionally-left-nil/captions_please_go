package twitter

import (
	"encoding/json"
	"fmt"
)

type Tweet struct {
	Id            string
	Text          string
	ParentTweetId string
	Type          TweetType
	Mentions      []Mention
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

type Mention struct {
	User
	StartIndex int `json:"start"`
	EndIndex   int `json:"end"`
}

type rawTweet struct {
	Id               string                  `json:"id_str"`
	FullText         *string                 `json:"full_text"`
	TruncatedText    *string                 `json:"text"`
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
	Text         *string `json:"full_text"`
	VisibleRange []int   `json:"display_text_range"`
}

type rawMention struct {
	User
	Indices []int `json:"indices"`
}

type media struct {
	Id      string  `json:"id_str"`
	Url     string  `json:"media_url_https"`
	Type    string  `json:"type"`
	AltText *string `json:"ext_alt_text"`
}

type entities struct {
	Mentions []rawMention `json:"user_mentions"`
	Media    []media      `json:"media"`
}

func (t *rawTweet) Text() (string, error) {
	var err error
	var text string
	if t.Truncated {
		et := t.ExtendedTweet
		if et == nil {
			err = invalidTweet("extended_tweet", nil)
		} else if et.Text == nil {
			err = invalidTweet("extended_tweet.full_text", et.Text)
		} else {
			err = validateRange(*et.Text, et.VisibleRange, "extended_tweet.full_text")
		}

		if err == nil {
			start, end := et.VisibleRange[0], et.VisibleRange[1]
			text = (*et.Text)[start:end]
		}
	} else if t.FullText != nil {
		err = validateRange(*t.FullText, t.VisibleRange, "full_text")
		if err == nil {
			start, end := t.VisibleRange[0], t.VisibleRange[1]
			text = (*t.FullText)[start:end]
		}
	} else if t.TruncatedText != nil {
		err = validateRange(*t.TruncatedText, t.VisibleRange, "text")
		if err == nil {
			start, end := t.VisibleRange[0], t.VisibleRange[1]
			text = (*t.TruncatedText)[start:end]
		}
	} else {
		err = invalidTweet("text", nil)
	}
	return text, err
}

func (tweet *rawTweet) TweetType() TweetType {

	if tweet.QuoteTweet {
		return QuoteTweet
	}
	if tweet.RetweetedStatus != nil {
		return Retweet
	}
	return SimpleTweet
}

func (t *rawTweet) Mentions() ([]Mention, error) {
	var err error
	var mentions []Mention

	if t.Entities != nil && t.Entities.Mentions != nil && len(t.Entities.Mentions) > 0 {
		mentions = []Mention{}
		var displayText *string
		for i, rawMention := range t.Entities.Mentions {
			if displayText == nil {
				var text string
				text, err = t.Text()
				if err == nil {
					displayText = &text
				}
			}

			if err == nil {
				err = validateRange(*displayText, rawMention.Indices, fmt.Sprintf("mention[%d]", i))
				if err == nil && rawMention.Id == "" {
					err = invalidTweet(fmt.Sprintf("mention[%d].Id", i), rawMention.Id)
				}
			}

			if err != nil {
				break
			}
			mention := Mention{User: rawMention.User, StartIndex: rawMention.Indices[0], EndIndex: rawMention.Indices[1]}
			mentions = append(mentions, mention)
		}
	}
	return mentions, err
}

func (t *rawTweet) Media() []media {
	if t.ExtendedEntities != nil && t.ExtendedEntities.Media != nil && len(t.ExtendedEntities.Media) > 0 {
		return t.ExtendedEntities.Media
	}
	return nil
}

func (t *Tweet) UnmarshalJSON(bytes []byte) error {
	raw := rawTweet{}
	err := json.Unmarshal(bytes, &raw)
	if err != nil {
		return err
	}
	t.Id = raw.Id
	if t.Id == "" {
		return invalidTweet("id", t.Id)
	}

	t.Text, err = raw.Text()
	if err != nil {
		return err
	}

	t.ParentTweetId = raw.ParentTweetId
	t.User = raw.User
	t.Type = raw.TweetType()

	t.Mentions, err = raw.Mentions()
	if err != nil {
		return err
	}
	t.Media = raw.Media()
	return nil
}

func validateRange(text string, indices []int, key string) error {
	if len(indices) != 2 {
		return invalidTweet(key+".len", indices)
	}
	start, end := indices[0], indices[1]
	if start < 0 || end > len(text) || start > end {
		return invalidTweet(key, indices)
	}
	return nil
}

func invalidTweet(key string, value interface{}) error {
	return fmt.Errorf("tweet[%s] has an invalid value %v", key, value)
}
