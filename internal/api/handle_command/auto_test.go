package handle_command

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	twitter_test "github.com/AnilRedshift/captions_please_go/pkg/twitter/test"
	"github.com/AnilRedshift/captions_please_go/pkg/vision"
	vision_test "github.com/AnilRedshift/captions_please_go/pkg/vision/test"
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
)

func TestWithAuto(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockTwitter := twitter_test.MockTwitter{T: t}
	ctx = WithAuto(ctx, &mockTwitter)
	state := getAutoState(ctx)
	assert.NotNil(t, state)
}

func TestHandleAuto(t *testing.T) {
	altText := "this photo has some cool alt text"
	user := twitter.User{Display: "Ada Bear", Id: "999", Username: "@ada_bear"}
	threeHundredChars := strings.Repeat("a", 300)

	onePhoto := []twitter.Media{{Type: "photo", Url: "photo.jpg", AltText: &altText}}
	oneVideo := []twitter.Media{{Type: "video", Url: "video.mp4"}}
	onePhotoWithoutAlt := []twitter.Media{{Type: "photo", Url: "photo.jpg"}}
	longPhoto := []twitter.Media{{Type: "photo", Url: threeHundredChars}}
	mixedMedia := []twitter.Media{{Type: "photo", Url: "photo1.jpg", AltText: &altText}, {Type: "video", Url: "video.mp4"}, {Type: "photo", Url: "photo2.jpg"}}
	tweetWithOnePhoto := twitter.Tweet{Id: "withOnePhoto", User: user, Media: onePhoto}
	tweetWithOneVideo := twitter.Tweet{Id: "withOneVideo", User: user, Media: oneVideo}
	tweetWithoutAlt := twitter.Tweet{Id: "onePhotoWithoutAlt", User: user, Media: onePhotoWithoutAlt}
	tweetWithMixedMedia := twitter.Tweet{Id: "withMixedMedia", User: user, Media: mixedMedia}
	tweetWithLongPhoto := twitter.Tweet{Id: "withOneLongPhoto", User: user, Media: longPhoto}
	tests := []struct {
		name        string
		tweet       *twitter.Tweet
		googleErr   error
		azureErr    error
		messages    []string
		confidences []float32
		hasErr      bool
	}{
		{
			name:     "Responds with the alt text if set",
			tweet:    &tweetWithOnePhoto,
			messages: []string{altText},
		},
		{
			name:        "Returns the alt text of mixed media",
			tweet:       &tweetWithMixedMedia,
			googleErr:   errors.New("dont be evil amiright"),
			confidences: []float32{0.8},
			messages:    []string{"Image 1: this photo has some cool alt text\nImage 3: photo2.jpg is so pretty(0.8)"},
		},
		{
			name:        "Returns both the OCR and a description for a short message",
			tweet:       &tweetWithoutAlt,
			confidences: []float32{0.8, 0.7},
			messages:    []string{"photo.jpg is so pretty(0.8). It might also be photo.jpg is so pretty(0.7). It contains the text: photo.jpg ocr response"},
		},
		{
			name:        "Returns just the OCR for a long message",
			tweet:       &tweetWithLongPhoto,
			confidences: []float32{0.8, 0.7},
			messages:    []string{threeHundredChars[:280], threeHundredChars[280:] + " ocr response"},
		},
		{
			name:        "Returns the description if the OCR fails",
			tweet:       &tweetWithoutAlt,
			googleErr:   errors.New("dont be evil amiright"),
			confidences: []float32{0.8, 0.7},
			messages:    []string{"photo.jpg is so pretty(0.8). It might also be photo.jpg is so pretty(0.7)"},
		},
		{
			name:        "Returns an error if both OCR and description fails",
			tweet:       &tweetWithoutAlt,
			googleErr:   errors.New("dont be evil amiright"),
			azureErr:    errors.New("did you try bing instead"),
			confidences: []float32{0.8, 0.7},
			messages:    []string{"I'm at a loss for words, sorry!"},
			hasErr:      true,
		},
		{
			name:     "Returns an error if no photos",
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

			mockGoogle := vision_test.MockGoogle{T: t, GetOCRMock: func(url string) (result *vision.OCRResult, err error) {
				ocr := vision.OCRResult{Text: url + " ocr response"}
				return &ocr, test.googleErr
			}}

			mockAzure := vision_test.MockAzure{T: t, DescribeMock: func(url string) ([]vision.VisionResult, error) {
				results := make([]vision.VisionResult, len(test.confidences))
				for i, confidence := range test.confidences {
					text := fmt.Sprintf("%s is so pretty(%.1f)", url, confidence)
					results[i] = vision.VisionResult{Text: text, Confidence: confidence}
				}
				return results, test.azureErr
			}}

			autoState := autoState{
				client: &mockTwitter,
			}

			ocrState := ocrState{
				client: &mockTwitter,
				google: &mockGoogle,
			}

			describeState := describeState{
				client:    &mockTwitter,
				describer: &mockAzure,
			}

			ctx = setOCRState(ctx, &ocrState)
			ctx = setAutoState(ctx, &autoState)
			ctx = WithAltText(ctx, &mockTwitter)
			ctx = setDescribeState(ctx, &describeState)
			ctx, err := replier.WithReplier(ctx, &mockTwitter)
			assert.NoError(t, err)

			result := HandleAuto(ctx, test.tweet)
			if test.hasErr {
				assert.Error(t, result.Err)
			} else {
				assert.NoError(t, result.Err)
			}
			assert.Equal(t, test.messages, sentMessages)
		})
	}
}
