package handle_command

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/AnilRedshift/captions_please_go/pkg/vision"
	vision_test "github.com/AnilRedshift/captions_please_go/pkg/vision/test"
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

func TestWithDescribe(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	secrets := &common.Secrets{GooglePrivateKeySecret: vision_test.DummyGoogleCert, AzureComputerVisionKey: "123"}
	ctx = common.SetSecrets(ctx, secrets)
	ctx, err := WithDescribe(ctx)
	assert.NoError(t, err)
	state := getDescriberState(ctx)
	assert.NotNil(t, state)
}

func TestWithDescribeHandlesGoogleFailure(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	secrets := &common.Secrets{GooglePrivateKeySecret: "a bad cert", AzureComputerVisionKey: "123"}
	ctx = common.SetSecrets(ctx, secrets)
	_, err := WithDescribe(ctx)
	assert.Error(t, err)

}

func TestGetDescribeMediaResponse(t *testing.T) {
	user := twitter.User{Display: "Ada Bear", Id: "999", Username: "@ada_bear"}

	onePhoto := []twitter.Media{{Type: "photo", Url: "photo.jpg"}}
	twoPhotos := []twitter.Media{{Type: "photo", Url: "photo1.jpg"}, {Type: "photo", Url: "photo2.jpg"}}
	oneVideo := []twitter.Media{{Type: "video", Url: "video.mp4"}}
	mixedMedia := []twitter.Media{{Type: "photo", Url: "photo.jpg"}, {Type: "video", Url: "video.mp4"}}
	tweetWithOnePhoto := twitter.Tweet{Id: "withOnePhoto", User: user, Media: onePhoto}
	tweetWithTwoPhotos := twitter.Tweet{Id: "withTwoPhotos", User: user, Media: twoPhotos}
	tweetWithMixedMedia := twitter.Tweet{Id: "withMixedMedia", User: user, Media: mixedMedia}
	tweetWithOneVideo := twitter.Tweet{Id: "withOneVideo", User: user, Media: oneVideo}

	wrongLangErr := structured_error.Wrap(errors.New("wrong language"), structured_error.UnsupportedLanguage)
	translateErr := structured_error.Wrap(errors.New("mais non"), structured_error.TranslateError)
	tests := []struct {
		name         string
		tweet        *twitter.Tweet
		lang         *language.Tag
		confidences  []float32
		azureErr     error
		translateErr error
		expected     []mediaResponse
	}{
		{
			name:        "Returns a single description",
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.8},
			expected:    []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Unlocalized("photo.jpg is so pretty(0.8)")}},
		},
		{
			name:        "Returns a single description in the users language",
			lang:        &language.Hindi,
			azureErr:    wrongLangErr,
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.8},
			expected:    []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Unlocalized("<translated photo.jpg is so pretty(0.8) />")}},
		},
		{
			name:        "Returns two descriptions for an image",
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.8, 0.6},
			expected:    []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Unlocalized("photo.jpg is so pretty(0.8). It might also be photo.jpg is so pretty(0.6)")}},
		},
		{
			name:        "Returns three descriptions for an image",
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.8, 0.6, 0.5},
			expected:    []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Unlocalized("photo.jpg is so pretty(0.8). It might also be photo.jpg is so pretty(0.6). It might also be photo.jpg is so pretty(0.5)")}},
		},
		{
			name:        "Responds with the description for two photos",
			tweet:       &tweetWithTwoPhotos,
			confidences: []float32{0.8, 0.7},
			expected: []mediaResponse{
				{index: 0, responseType: foundVisionResponse, reply: message.Unlocalized("photo1.jpg is so pretty(0.8). It might also be photo1.jpg is so pretty(0.7)")},
				{index: 1, responseType: foundVisionResponse, reply: message.Unlocalized("photo2.jpg is so pretty(0.8). It might also be photo2.jpg is so pretty(0.7)")},
			},
		},
		{
			name:        "Responds with the description of a photo, ignoring non-photos",
			tweet:       &tweetWithMixedMedia,
			confidences: []float32{0.8},
			expected: []mediaResponse{
				{index: 0, responseType: foundVisionResponse, reply: message.Unlocalized("photo.jpg is so pretty(0.8)")},
				{index: 1, responseType: doNothingResponse},
			},
		},
		{
			name:        "Ignores low confidence suggestions",
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.8, 0.1},
			expected:    []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Unlocalized("photo.jpg is so pretty(0.8)")}},
		},
		{
			name:         "Returns unsupported error message if translating fails",
			lang:         &language.Hindi,
			azureErr:     wrongLangErr,
			translateErr: translateErr,
			tweet:        &tweetWithOnePhoto,
			confidences:  []float32{0.8},
			expected:     []mediaResponse{{index: 0, responseType: foundVisionResponse, err: translateErr}},
		},
		{
			name:        "Returns error message if everything is low-confidence",
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.1, 0.1},
			expected:    []mediaResponse{{index: 0, responseType: foundVisionResponse, err: structured_error.Wrap(errors.New("low confidences"), structured_error.DescribeError)}},
		},
		{
			name:        "Returns error message if azure fails",
			tweet:       &tweetWithOnePhoto,
			confidences: []float32{0.8},
			azureErr:    errors.New("Lock the taskbar, lock the taskbar"),
			expected:    []mediaResponse{{index: 0, responseType: foundVisionResponse, err: structured_error.Wrap(errors.New("lock the taskbar, lock the taskbar"), structured_error.DescribeError)}},
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

			mockAzure := vision_test.MockAzure{T: t, DescribeMock: func(url string) ([]vision.VisionResult, error) {
				results := make([]vision.VisionResult, len(test.confidences))
				for i, confidence := range test.confidences {
					text := fmt.Sprintf("%s is so pretty(%.1f)", url, confidence)
					results[i] = vision.VisionResult{Text: text, Confidence: confidence}
				}
				return results, test.azureErr
			}}

			mockGoogle := vision_test.MockGoogle{T: t, TranslateMock: func(message string) (language.Tag, string, error) {
				translated := "<translated " + message + " />"
				return language.English, translated, test.translateErr
			}}

			state := describeState{
				describer:  &mockAzure,
				translator: &mockGoogle,
			}

			ctx = setDescribeState(ctx, &state)

			if test.lang != nil {
				ctx = message.WithLanguage(ctx, *test.lang)
			}
			result := getDescribeMediaResponse(ctx, test.tweet)
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
