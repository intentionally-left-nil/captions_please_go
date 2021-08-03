package handle_command

import (
	"context"
	"errors"
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

func TestWithOCR(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	secrets := &common.Secrets{GooglePrivateKeySecret: vision_test.DummyGoogleCert}
	ctx = common.SetSecrets(ctx, secrets)
	mockTwitter := twitter_test.MockTwitter{T: t}
	ctx, err := WithOCR(ctx, &mockTwitter)
	assert.NoError(t, err)
	state := getOCRState(ctx)
	assert.NotNil(t, state)
}

func TestHandleOCR(t *testing.T) {

	user := twitter.User{Display: "Ada Bear", Id: "999", Username: "@ada_bear"}
	onePhoto := []twitter.Media{{Type: "photo", Url: "photo.jpg"}}
	oneVideo := []twitter.Media{{Type: "video", Url: "video.mp4"}}
	twoPhotos := []twitter.Media{{Type: "photo", Url: "photo1.jpg"}, {Type: "photo", Url: "photo2.jpg"}}
	mixedMedia := []twitter.Media{{Type: "photo", Url: "photo.jpg"}, {Type: "video", Url: "video.mp4"}}
	tweetWithOnePhoto := twitter.Tweet{Id: "withOnePhoto", User: user, Media: onePhoto}
	tweetWithOneVideo := twitter.Tweet{Id: "withOneVideo", User: user, Media: oneVideo}
	tweetWithTwoPhotos := twitter.Tweet{Id: "withTwoPhotos", User: user, Media: twoPhotos}
	tweetWithMixedMedia := twitter.Tweet{Id: "withMixedMedia", User: user, Media: mixedMedia}
	tests := []struct {
		name      string
		tweet     *twitter.Tweet
		googleErr error
		messages  []string
		hasErr    bool
	}{
		{
			name:     "Responds with the OCR of a single image",
			tweet:    &tweetWithOnePhoto,
			messages: []string{"ocr response for photo.jpg"},
		},
		{
			name:      "Responds with an error if OCR fails",
			tweet:     &tweetWithTwoPhotos,
			googleErr: errors.New("google fired another good engineer now their code is broken"),
			messages:  []string{"Image 1: I encountered difficulties scanning the image. Sorry!\nImage 2: I encountered difficulties scanning the image. Sorry!"},
			hasErr:    true,
		},
		{
			name:     "Responds with the OCR of multiple images",
			tweet:    &tweetWithTwoPhotos,
			messages: []string{"Image 1: ocr response for photo1.jpg\nImage 2: ocr response for photo2.jpg"},
		},
		{
			name:     "Responds with the OCR for mixed media, ignoring non-photos",
			tweet:    &tweetWithMixedMedia,
			messages: []string{"ocr response for photo.jpg"},
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

			mockGoogle := vision_test.MockGoogle{T: t, GetOCRMock: func(url string) (result *vision.OCRResult, err error) {
				ocr := vision.OCRResult{Text: "ocr response for " + url}
				return &ocr, test.googleErr
			}}

			state := ocrState{
				client: &mockTwitter,
				google: &mockGoogle,
			}
			ctx = setOCRState(ctx, &state)
			ctx, err := replier.WithReplier(ctx, &mockTwitter)
			assert.NoError(t, err)
			result := <-HandleOCR(ctx, test.tweet)

			if test.hasErr {
				assert.Error(t, result.Err)
			} else {
				assert.NoError(t, result.Err)
			}
			assert.Equal(t, test.messages, sentMessages)
		})
	}
}
