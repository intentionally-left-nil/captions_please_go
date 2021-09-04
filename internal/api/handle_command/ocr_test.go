package handle_command

import (
	"context"
	"errors"
	"testing"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/AnilRedshift/captions_please_go/pkg/vision"
	vision_test "github.com/AnilRedshift/captions_please_go/pkg/vision/test"
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithOCR(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	secrets := &common.Secrets{GooglePrivateKeySecret: vision_test.DummyGoogleCert}
	ctx = common.SetSecrets(ctx, secrets)
	ctx, err := WithOCR(ctx)
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

	googleErr := errors.New("google fired another good engineer now their code is broken")
	tests := []struct {
		name      string
		tweet     *twitter.Tweet
		googleErr error
		expected  []mediaResponse
	}{
		{
			name:     "Responds with the OCR of a single image",
			tweet:    &tweetWithOnePhoto,
			expected: []mediaResponse{{index: 0, responseType: foundOCRResponse, reply: "ocr response for photo.jpg"}},
		},
		{
			name:      "Responds with an error if OCR fails",
			tweet:     &tweetWithTwoPhotos,
			googleErr: googleErr,
			expected: []mediaResponse{
				{index: 0, responseType: foundOCRResponse, err: structured_error.Wrap(googleErr, structured_error.OCRError)},
				{index: 1, responseType: foundOCRResponse, err: structured_error.Wrap(googleErr, structured_error.OCRError)},
			},
		},
		{
			name:  "Responds with the OCR of multiple images",
			tweet: &tweetWithTwoPhotos,
			expected: []mediaResponse{
				{index: 0, responseType: foundOCRResponse, reply: "ocr response for photo1.jpg"},
				{index: 1, responseType: foundOCRResponse, reply: "ocr response for photo2.jpg"},
			},
		},
		{
			name:  "Responds with the OCR for mixed media, ignoring non-photos",
			tweet: &tweetWithMixedMedia,
			expected: []mediaResponse{
				{index: 0, responseType: foundOCRResponse, reply: "ocr response for photo.jpg"},
				{index: 1, responseType: doNothingResponse}},
		},
		{
			name:     "Lets the user know there aren't any photos to decode",
			tweet:    &tweetWithOneVideo,
			expected: []mediaResponse{{index: 0, responseType: doNothingResponse}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer leaktest.Check(t)()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockGoogle := vision_test.MockGoogle{T: t, GetOCRMock: func(url string) (result *vision.OCRResult, err error) {
				ocr := vision.OCRResult{Text: "ocr response for " + url}
				return &ocr, test.googleErr
			}}

			state := ocrState{
				google: &mockGoogle,
			}
			ctx = setOCRState(ctx, &state)
			result := getOCRMediaResponse(ctx, test.tweet)
			assert.Equal(t, len(test.expected), len(result))
			for i, expectedMessage := range test.expected {
				if expectedMessage.err == nil {
					assert.Equal(t, expectedMessage, result[i])
				} else {
					require.Error(t, result[i].err)
					assert.Equal(t, expectedMessage.err.Type(), result[i].err.Type())
					assert.Equal(t, expectedMessage.index, result[i].index)
					assert.Equal(t, expectedMessage.reply, result[i].reply)
					assert.Equal(t, expectedMessage.responseType, result[i].responseType)
				}
			}
		})
	}
}
