package twitter

import (
	"encoding/json"
	"fmt"
)

type Tweet struct {
	Id                string
	FullText          string
	VisibleText       string
	VisibleTextOffset int
	ParentTweetId     string
	Type              TweetType
	Mentions          []Mention
	User              User
	Media             []Media
	FallbackMedia     []Media
	QuoteTweet        *Tweet
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
	Visible    bool
}

type rawTweet struct {
	Id                string                  `json:"id_str"`
	FullText          *string                 `json:"full_text"`
	TruncatedText     *string                 `json:"text"`
	VisibleRange      *[]int                  `json:"display_text_range"`
	ExtendedTweet     *extendedTweet          `json:"extended_tweet"`
	Truncated         bool                    `json:"truncated"`
	ParentTweetId     string                  `json:"in_reply_to_status_id_str"`
	IsQuoteTweet      bool                    `json:"is_quote_status"`
	QuoteTweet        *Tweet                  `json:"quoted_status"`
	RetweetedStatus   *map[string]interface{} `json:"retweeted_status"`
	TruncatedEntities *entities               `json:"entities"`
	ExtendedEntities  *entities               `json:"extended_entities"`
	User              User                    `json:"user"`
}

type extendedTweet struct {
	Text             *string   `json:"full_text"`
	VisibleRange     []int     `json:"display_text_range"`
	Entities         *entities `json:"entities"`
	ExtendedEntities *entities `json:"extended_entities"`
}

type rawMention struct {
	User
	Indices []int `json:"indices"`
}

type rawUrl struct {
	Url     string `json:"expanded_url"`
	Indices []int  `json:"indices"`
}

type Media struct {
	Id      string  `json:"id_str"`
	Url     string  `json:"media_url_https"`
	Type    string  `json:"type"`
	AltText *string `json:"ext_alt_text"`
}

type entities struct {
	Mentions []rawMention `json:"user_mentions"`
	Media    []Media      `json:"media"`
	Urls     []rawUrl     `json:"urls"`
}

type textInfo struct {
	full  string
	start int
	end   int
}

func (ti textInfo) Visible() string {
	return ti.full[ti.start:ti.end]
}

func (t *rawTweet) TextInfo() (textInfo, error) {
	var err error
	info := textInfo{}
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
			info = textInfo{full: *et.Text, start: et.VisibleRange[0], end: et.VisibleRange[1]}
		}
	} else if t.FullText != nil {
		var visibleRange []int
		if t.VisibleRange == nil {
			visibleRange = []int{}
		} else {
			visibleRange = *t.VisibleRange
		}
		err = validateRange(*t.FullText, visibleRange, "full_text")
		if err == nil {
			info = textInfo{full: *t.FullText, start: visibleRange[0], end: visibleRange[1]}
		}
	} else if t.TruncatedText != nil {
		var visibleRange []int
		if t.VisibleRange == nil {
			// The VisibleRange isn't always present for the fallback text field
			// Just take the whole length if missing
			visibleRange = []int{0, len(*t.TruncatedText)}
		} else {
			visibleRange = *t.VisibleRange
		}

		err = validateRange(*t.TruncatedText, visibleRange, "text")
		if err == nil {
			info = textInfo{full: *t.TruncatedText, start: visibleRange[0], end: visibleRange[1]}
		}
	} else {
		err = invalidTweet("text", nil)
	}

	entities := t.Entities()
	if err == nil && t.TweetType() == QuoteTweet && entities != nil {
		// Quote tweets include the url at the end of the tweet
		// but it's not visible to the user. Remove it from the visible range.
		urlStart := -1
		for i, url := range entities.Urls {
			if validateRange(info.full, url.Indices, fmt.Sprintf("entities.urls[%d]", i)) == nil &&
				url.Indices[0] > urlStart {
				urlStart = url.Indices[0]
			}
		}

		if urlStart >= 0 && urlStart >= info.start {
			// This will keep the trailing space before the URL but it's not worth optimizing for
			// as downstream callers will just strip excess space
			info.end = urlStart
		}
	}
	return info, err
}

func (tweet *rawTweet) Entities() *entities {
	var entities *entities
	if tweet.Truncated && tweet.ExtendedTweet != nil {
		entities = tweet.ExtendedTweet.Entities
	} else if !tweet.Truncated {
		entities = tweet.TruncatedEntities
	}
	return entities
}

func (tweet *rawTweet) TweetType() TweetType {

	// A retweet of a quote tweet has both IsQuoteTweet=true !?! and a RetweetedStatus
	// A plain quote tweet just has IsQuoteTweet set to true. tl;dr: check IsQuoteTweet first
	if tweet.RetweetedStatus != nil {
		return Retweet
	}
	if tweet.IsQuoteTweet {
		return QuoteTweet
	}
	return SimpleTweet
}

func (t *rawTweet) Mentions() ([]Mention, error) {
	var err error
	var mentions []Mention
	entities := t.Entities()
	if entities != nil && entities.Mentions != nil && len(entities.Mentions) > 0 {
		mentions = []Mention{}
		var textInfo textInfo
		textInfo, err = t.TextInfo()
		if err == nil {
			for i, rawMention := range entities.Mentions {

				if err == nil {
					err = validateRange(textInfo.full, rawMention.Indices, fmt.Sprintf("mention[%d]", i))
					if err == nil && rawMention.Id == "" {
						err = invalidTweet(fmt.Sprintf("mention[%d].Id", i), rawMention.Id)
					}
				}

				if err != nil {
					break
				}
				start, end := rawMention.Indices[0], rawMention.Indices[1]
				isVisible := start >= textInfo.start && end <= textInfo.end
				mention := Mention{
					User:       rawMention.User,
					StartIndex: start,
					EndIndex:   end,
					Visible:    isVisible}
				mentions = append(mentions, mention)
			}
		}
	}
	return mentions, err
}

func (t *rawTweet) Media() (media []Media) {
	if t.ExtendedTweet != nil && t.ExtendedTweet.ExtendedEntities != nil && len(t.ExtendedTweet.ExtendedEntities.Media) > 0 {
		media = t.ExtendedTweet.ExtendedEntities.Media
	} else if t.ExtendedEntities != nil && t.ExtendedEntities.Media != nil && len(t.ExtendedEntities.Media) > 0 {
		media = t.ExtendedEntities.Media
	}
	return media
}

func (t *rawTweet) FallbackMedia() []Media {
	if t.TruncatedEntities != nil && t.TruncatedEntities.Media != nil && len(t.TruncatedEntities.Media) > 0 {
		return t.TruncatedEntities.Media
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

	var ti textInfo
	ti, err = raw.TextInfo()
	if err != nil {
		return err
	}
	t.VisibleText = ti.Visible()
	t.FullText = ti.full
	t.VisibleTextOffset = ti.start

	t.ParentTweetId = raw.ParentTweetId
	t.User = raw.User
	t.Type = raw.TweetType()

	t.QuoteTweet = raw.QuoteTweet

	t.Mentions, err = raw.Mentions()
	if err != nil {
		return err
	}
	t.Media = raw.Media()
	t.FallbackMedia = raw.FallbackMedia()
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
