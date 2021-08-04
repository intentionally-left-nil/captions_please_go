package handle_command

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	twitter_test "github.com/AnilRedshift/captions_please_go/pkg/twitter/test"
	"github.com/AnilRedshift/captions_please_go/pkg/vision"
	vision_test "github.com/AnilRedshift/captions_please_go/pkg/vision/test"
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
)

func TestWithDescribe(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	secrets := &common.Secrets{AzureComputerVisionKey: "123"}
	ctx = common.SetSecrets(ctx, secrets)
	mockTwitter := twitter_test.MockTwitter{T: t}
	ctx = WithDescribe(ctx, &mockTwitter)
	state := getDescriberState(ctx)
	assert.NotNil(t, state)
}

func TestHandleDescribe(t *testing.T) {
	threeHundredChars := strings.Repeat("a", 300)

	user := twitter.User{Display: "Ada Bear", Id: "999", Username: "@ada_bear"}

	onePhoto := []twitter.Media{{Type: "photo", Url: "photo.jpg"}}
	longPhoto := []twitter.Media{{Type: "photo", Url: threeHundredChars}}
	twoPhotos := []twitter.Media{{Type: "photo", Url: "photo1.jpg"}, {Type: "photo", Url: "photo2.jpg"}}
	oneVideo := []twitter.Media{{Type: "video", Url: "video.mp4"}}
	mixedMedia := []twitter.Media{{Type: "photo", Url: "photo.jpg"}, {Type: "video", Url: "video.mp4"}}
	tweetWithOnePhoto := twitter.Tweet{Id: "withOnePhoto", User: user, Media: onePhoto}
	tweetWithLongPhoto := twitter.Tweet{Id: "withOneLongPhoto", User: user, Media: longPhoto}
	tweetWithTwoPhotos := twitter.Tweet{Id: "withTwoPhotos", User: user, Media: twoPhotos}
	tweetWithMixedMedia := twitter.Tweet{Id: "withMixedMedia", User: user, Media: mixedMedia}
	tweetWithOneVideo := twitter.Tweet{Id: "withOneVideo", User: user, Media: oneVideo}
	tests := []struct {
		name        string
		tweet       *twitter.Tweet
		confidences []float32
		azureErr    error
		messages    []string
		hasErr      bool
	}{
		{
			name:        "Returns a single description",
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.8},
			messages:    []string{"photo.jpg is so pretty(0.8)"},
		},
		{
			name:        "Splits a long description into multiple tweets",
			tweet:       &tweetWithLongPhoto,
			confidences: []float32{0.8},
			messages:    []string{threeHundredChars[:280], threeHundredChars[280:] + " is so pretty(0.8)"},
		},
		{
			name:        "Returns two descriptions for an image",
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.8, 0.6},
			messages:    []string{"photo.jpg is so pretty(0.8). It might also be photo.jpg is so pretty(0.6)"},
		},
		{
			name:        "Returns three descriptions for an image",
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.8, 0.6, 0.5},
			messages:    []string{"photo.jpg is so pretty(0.8). It might also be photo.jpg is so pretty(0.6). It might also be photo.jpg is so pretty(0.5)"},
		},
		{
			name:        "Responds with the description for two photos",
			tweet:       &tweetWithTwoPhotos,
			confidences: []float32{0.8, 0.7},
			messages:    []string{"Image 1: photo1.jpg is so pretty(0.8). It might also be photo1.jpg is so pretty(0.7)\nImage 2: photo2.jpg is so pretty(0.8). It might also be photo2.jpg is so pretty(0.7)"},
		},
		{
			name:        "Responds with the description of a photo, ignoring non-photos",
			tweet:       &tweetWithMixedMedia,
			confidences: []float32{0.8},
			messages:    []string{"photo.jpg is so pretty(0.8)"},
		},
		{
			name:        "Ignores low confidence suggestions",
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.8, 0.1},
			messages:    []string{"photo.jpg is so pretty(0.8)"},
		},
		{
			name:        "Returns error message if everything is low-confidence",
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.1, 0.1},
			messages:    []string{"I'm at a loss for words, sorry!"},
			hasErr:      true,
		},
		{
			name:        "Returns error message if azure fails",
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.8},
			azureErr:    errors.New("Lock the taskbar, lock the taskbar"),
			messages:    []string{"I'm at a loss for words, sorry!"},
			hasErr:      true,
		},
		{
			name:     "Lets the user know there aren't any photos to decode",
			tweet:    &tweetWithOneVideo,
			messages: []string{"I only know how to interpret photos right now, sorry!"},
			hasErr:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer leaktest.Check(t)()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sentMessages := []string{}
			mockTwitter := twitter_test.MockTwitter{T: t, TweetReplyMock: func(tweetID, message string) (*twitter.Tweet, error) {
				tweet := twitter.Tweet{Id: "123"}
				sentMessages = append(sentMessages, message)
				return &tweet, nil
			}}

			mockAzure := vision_test.MockAzure{T: t, DescribeMock: func(url string) ([]vision.VisionResult, error) {
				results := make([]vision.VisionResult, len(test.confidences))
				for i, confidence := range test.confidences {
					text := fmt.Sprintf("%s is so pretty(%.1f)", url, confidence)
					results[i] = vision.VisionResult{Text: text, Confidence: confidence}
				}
				return results, test.azureErr
			}}

			state := describeState{
				client:    &mockTwitter,
				describer: &mockAzure,
			}

			ctx = setDescribeState(ctx, &state)
			ctx, err := replier.WithReplier(ctx, &mockTwitter)
			assert.NoError(t, err)
			result := HandleDescribe(ctx, test.tweet)
			if test.hasErr {
				assert.Error(t, result.Err)
			} else {
				assert.NoError(t, result.Err)
			}
			assert.Equal(t, test.messages, sentMessages)
		})
	}
}
