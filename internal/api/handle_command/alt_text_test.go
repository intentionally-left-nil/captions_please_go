package handle_command

import (
	"context"
	"errors"
	"testing"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	vision_test "github.com/AnilRedshift/captions_please_go/pkg/vision/test"
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

func TestWithAltText(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	secrets := &common.Secrets{GooglePrivateKeySecret: vision_test.DummyGoogleCert, AzureComputerVisionKey: "123"}
	ctx = common.SetSecrets(ctx, secrets)
	ctx, err := WithAltText(ctx)
	assert.NoError(t, err)
	state := ctx.Value(theAltTextCtxKey)
	assert.NotNil(t, state)

}

func TestGetAltTextMediaResponse(t *testing.T) {
	user := twitter.User{Display: "Ada Bear", Id: "999", Username: "@ada_bear"}
	altText := "hello alt text"
	onePhotoWithAltText := []twitter.Media{{Type: "photo", AltText: &altText}}
	onePhotoWithoutAltText := []twitter.Media{{Type: "photo"}}
	mixedMedia := []twitter.Media{{Type: "photo"}, {Type: "photo", AltText: &altText}, {Type: "video"}}

	tweetWithoutAltText := twitter.Tweet{Id: "withoutMedia", Media: onePhotoWithoutAltText, User: user}
	tweetWithAltText := twitter.Tweet{Id: "withAltText", Media: onePhotoWithAltText, User: user}
	tweetWithMixedMedia := twitter.Tweet{Id: "withAltText", Media: mixedMedia, User: user}
	tweetWithoutMedia := twitter.Tweet{Id: "NoMedia"}
	tests := []struct {
		name         string
		command      command
		tweet        *twitter.Tweet
		translateErr error
		expected     []mediaResponse
	}{
		{
			name:     "Responds with the provided alt_text of a single image",
			tweet:    &tweetWithAltText,
			expected: []mediaResponse{{index: 0, responseType: foundAltTextResponse, reply: message.Unlocalized(altText)}},
		},
		{
			name:     "Translates the alt text into the desired language",
			tweet:    &tweetWithAltText,
			command:  command{altText: true, translate: true},
			expected: []mediaResponse{{index: 0, responseType: foundAltTextResponse, reply: message.Unlocalized("<translated hello alt text />")}},
		},
		{
			name:         "Ignores translation errors",
			command:      command{altText: true, translate: true},
			tweet:        &tweetWithAltText,
			translateErr: errors.New("Google is confused, again"),
			expected:     []mediaResponse{{index: 0, responseType: foundAltTextResponse, reply: message.Unlocalized(altText)}},
		},
		{
			name:     "Responds with no alt text when missing",
			tweet:    &tweetWithoutAltText,
			expected: []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: "Ada Bear didn't provide any alt text when posting the image"}},
		},
		{
			name:  "Responds to a tweet with multiple images",
			tweet: &tweetWithMixedMedia,
			expected: []mediaResponse{
				{index: 0, responseType: missingAltTextResponse, reply: "Ada Bear didn't provide any alt text when posting the image"},
				{index: 1, responseType: foundAltTextResponse, reply: "hello alt text"},
				{index: 2, responseType: doNothingResponse}},
		},
		{
			name:     "Does nothing if there's no media",
			tweet:    &tweetWithoutMedia,
			expected: []mediaResponse{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer leaktest.Check(t)()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockGoogle := vision_test.MockGoogle{T: t, TranslateMock: func(message string) (language.Tag, string, error) {
				translated := "<translated " + message + " />"
				return language.English, translated, test.translateErr
			}}

			state := &altTextState{translator: &mockGoogle}
			ctx = context.WithValue(ctx, theAltTextCtxKey, state)

			result := getAltTextMediaResponse(ctx, test.command, test.tweet)
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
