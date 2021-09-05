package handle_command

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	twitter_test "github.com/AnilRedshift/captions_please_go/pkg/twitter/test"
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleCommand(t *testing.T) {
	someText := "buffalo buffalo etc. etc."
	otherText := "this text too"
	noAltText := "no alt text hiss"
	anErr := structured_error.Wrap(errors.New("something bad"), structured_error.Unknown)
	blockedErr := structured_error.Wrap(errors.New("something bad"), structured_error.UserBlockedBot)
	noPhotosFoundErr := structured_error.Wrap(errors.New(""), structured_error.NoPhotosFound)
	ocrErr := structured_error.Wrap(errors.New("no results"), structured_error.OCRError)
	describeErr := structured_error.Wrap(errors.New("no results"), structured_error.DescribeError)
	tests := []struct {
		name         string
		command      command
		altText      []mediaResponse
		ocr          []mediaResponse
		description  []mediaResponse
		replyErr     structured_error.StructuredError
		findTweetErr structured_error.StructuredError
		expected     string
		hasErr       bool
	}{
		{
			name:     "replies with help",
			command:  command{help: true},
			expected: string(message.HelpMessage(context.Background())),
		},
		{
			name:     "silently ignores help failures",
			command:  command{help: true},
			expected: string(message.HelpMessage(context.Background())),
			replyErr: anErr,
			hasErr:   false,
		},
		{
			name:     "replies with unknown",
			command:  command{unknown: true},
			expected: string(message.UnknownCommandMessage(context.Background())),
		},
		{
			name:     "silently ignores failures sending the unknown command",
			command:  command{unknown: true},
			expected: string(message.UnknownCommandMessage(context.Background())),
			replyErr: anErr,
			hasErr:   false,
		},
		{
			name:         "replies with an error if finding the media fails",
			command:      command{altText: true},
			expected:     string(message.ErrorMessage(context.Background(), blockedErr)),
			findTweetErr: blockedErr,
			hasErr:       true,
		},
		{
			name:     "returns the alt text if found",
			command:  command{altText: true},
			altText:  []mediaResponse{{index: 0, responseType: foundAltTextResponse, reply: message.Localized(someText)}},
			expected: someText,
		},
		{
			name:     "informs the user if the alt text was not provided",
			command:  command{altText: true},
			altText:  []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			expected: noAltText,
		},
		{
			name:    "returns the alt text for multiple images",
			command: command{altText: true},
			altText: []mediaResponse{
				{index: 0, responseType: foundAltTextResponse, reply: message.Localized(someText)},
				{index: 1, responseType: foundAltTextResponse, reply: message.Localized(otherText)},
			},
			expected: "Image 1: buffalo buffalo etc. etc.\nImage 2: this text too",
		},
		{
			name:    "returns the alt text for a tweet with one alt text, one video, and one missing alt text",
			command: command{altText: true},
			altText: []mediaResponse{
				{index: 0, responseType: foundAltTextResponse, reply: message.Localized(someText)},
				{index: 1, responseType: doNothingResponse},
				{index: 2, responseType: missingAltTextResponse, reply: message.Localized(noAltText)},
			},
			expected: "Image 1: buffalo buffalo etc. etc.\nImage 3: no alt text hiss",
		},
		{
			name:     "informs the user no photos were found",
			command:  command{altText: true},
			altText:  []mediaResponse{{index: 0, responseType: doNothingResponse}},
			expected: string(message.ErrorMessage(context.Background(), noPhotosFoundErr)),
		},
		{
			name:     "returns the ocr if found",
			command:  command{ocr: true},
			ocr:      []mediaResponse{{index: 0, responseType: foundOCRResponse, reply: message.Localized(someText)}},
			expected: someText,
		},
		{
			name:     "informs the user no OCR could be computed",
			command:  command{ocr: true},
			ocr:      []mediaResponse{{index: 0, responseType: foundOCRResponse, err: ocrErr}},
			expected: "I'm at a loss for words, sorry!",
			// TODO: should this be considered an error?
			hasErr: true,
		},
		{
			name:    "returns the ocr for multiple images",
			command: command{ocr: true},
			ocr: []mediaResponse{
				{index: 0, responseType: foundOCRResponse, reply: message.Localized(someText)},
				{index: 1, responseType: foundOCRResponse, reply: message.Localized(otherText)},
			},
			expected: "Image 1: buffalo buffalo etc. etc.\nImage 2: this text too",
		},
		{
			name:    "returns ocr text for a tweet with one image, one video, and one photo without ocr",
			command: command{ocr: true},
			ocr: []mediaResponse{
				{index: 0, responseType: foundOCRResponse, reply: message.Localized(someText)},
				{index: 1, responseType: doNothingResponse},
				{index: 2, responseType: foundOCRResponse, err: ocrErr},
			},
			expected: "Image 1: buffalo buffalo etc. etc.\nImage 3: I'm at a loss for words, sorry!",
			// TODO: should this be considered an error?
			hasErr: true,
		},

		{
			name:        "returns the description if found",
			command:     command{describe: true},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Localized(someText)}},
			expected:    someText,
		},
		{
			name:        "informs the user no description could be computed",
			command:     command{describe: true},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, err: describeErr}},
			expected:    "I'm at a loss for words, sorry!",
			// TODO: should this be considered an error?
			hasErr: true,
		},
		{
			name:    "returns the description for multiple images",
			command: command{describe: true},
			description: []mediaResponse{
				{index: 0, responseType: foundVisionResponse, reply: message.Localized(someText)},
				{index: 1, responseType: foundVisionResponse, reply: message.Localized(otherText)},
			},
			expected: "Image 1: buffalo buffalo etc. etc.\nImage 2: this text too",
		},
		{
			name:    "returns description for a tweet with one image, one video, and one photo without a description",
			command: command{describe: true},
			description: []mediaResponse{
				{index: 0, responseType: foundVisionResponse, reply: message.Localized(someText)},
				{index: 1, responseType: doNothingResponse},
				{index: 2, responseType: foundVisionResponse, err: describeErr},
			},
			expected: "Image 1: buffalo buffalo etc. etc.\nImage 3: I'm at a loss for words, sorry!",
			// TODO: should this be considered an error?
			hasErr: true,
		},
		{
			name:     "alt text and OCR for an image",
			command:  command{altText: true, ocr: true},
			altText:  []mediaResponse{{index: 0, responseType: foundAltTextResponse, reply: message.Localized(someText)}},
			ocr:      []mediaResponse{{index: 0, responseType: foundOCRResponse, reply: message.Localized("my cool ocr")}},
			expected: "UserPostingMedia says it's buffalo buffalo etc. etc.. It contains the text: my cool ocr",
		},
		{
			name:     "ignores the OCR error if there's alt text",
			command:  command{altText: true, ocr: true},
			altText:  []mediaResponse{{index: 0, responseType: foundAltTextResponse, reply: message.Localized(someText)}},
			ocr:      []mediaResponse{{index: 0, responseType: foundOCRResponse, err: ocrErr}},
			expected: someText,
		},
		{
			name:     "No alt text and an OCR error",
			command:  command{altText: true, ocr: true},
			altText:  []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			ocr:      []mediaResponse{{index: 0, responseType: foundOCRResponse, err: ocrErr}},
			expected: "I'm at a loss for words, sorry!",
			hasErr:   true,
		},
		{
			name:     "OCR but no alt text for an image",
			command:  command{altText: true, ocr: true},
			altText:  []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			ocr:      []mediaResponse{{index: 0, responseType: foundOCRResponse, reply: message.Localized("my cool ocr")}},
			expected: "no alt text hiss. It contains the text: my cool ocr",
		},
		{
			name:        "alt text and description for an image",
			command:     command{altText: true, describe: true},
			altText:     []mediaResponse{{index: 0, responseType: foundAltTextResponse, reply: message.Localized(someText)}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Localized("my cool description")}},
			expected:    "UserPostingMedia says it's buffalo buffalo etc. etc.. I think it's my cool description",
		},
		{
			name:        "ignores the description error if there's alt text",
			command:     command{altText: true, describe: true},
			altText:     []mediaResponse{{index: 0, responseType: foundAltTextResponse, reply: message.Localized(someText)}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, err: describeErr}},
			expected:    someText,
		},
		{
			name:        "No alt text and a description error",
			command:     command{altText: true, describe: true},
			altText:     []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, err: describeErr}},
			expected:    "I'm at a loss for words, sorry!",
			hasErr:      true,
		},
		{
			name:        "description but no alt text for an image",
			command:     command{altText: true, describe: true},
			altText:     []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Localized("my cool description")}},
			expected:    "no alt text hiss. I think it's my cool description",
		},
		{
			name:        "OCR and a description for an image",
			command:     command{ocr: true, describe: true},
			ocr:         []mediaResponse{{index: 0, responseType: foundOCRResponse, reply: message.Localized("my cool ocr")}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Localized("my cool description")}},
			expected:    "my cool description. It contains the text: my cool ocr",
		},
		{
			name:        "alt text, OCR, and a description for an image",
			command:     command{altText: true, ocr: true, describe: true},
			altText:     []mediaResponse{{index: 0, responseType: foundAltTextResponse, reply: message.Localized(someText)}},
			ocr:         []mediaResponse{{index: 0, responseType: foundOCRResponse, reply: message.Localized("my cool ocr")}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Localized("my cool description")}},
			expected:    "UserPostingMedia says it's buffalo buffalo etc. etc.. I think it's my cool description. It contains the text: my cool ocr",
		},
		{
			name:        "no alt text, OCR, and a description for an image",
			command:     command{altText: true, ocr: true, describe: true},
			altText:     []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			ocr:         []mediaResponse{{index: 0, responseType: foundOCRResponse, reply: message.Localized("my cool ocr")}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Localized("my cool description")}},
			expected:    "no alt text hiss. I think it's my cool description. It contains the text: my cool ocr",
		},
		{
			name:        "no alt text, and donothings for the ocr and description",
			command:     command{altText: true, ocr: true, describe: true},
			altText:     []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			ocr:         []mediaResponse{{index: 0, responseType: doNothingResponse}},
			description: []mediaResponse{{index: 0, responseType: doNothingResponse}},
			expected:    noAltText,
		},
		{
			name:        "auto when there's just alt text",
			command:     command{auto: true},
			altText:     []mediaResponse{{index: 0, responseType: foundAltTextResponse, reply: message.Localized(someText)}},
			ocr:         []mediaResponse{{index: 0, responseType: doNothingResponse}},
			description: []mediaResponse{{index: 0, responseType: doNothingResponse}},
			expected:    someText,
		},
		{
			name:        "auto prefers the alt text even when other pieces are present",
			command:     command{auto: true},
			altText:     []mediaResponse{{index: 0, responseType: foundAltTextResponse, reply: message.Localized(someText)}},
			ocr:         []mediaResponse{{index: 0, responseType: foundOCRResponse, reply: message.Localized("my cool ocr")}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Localized("my cool description")}},
			expected:    someText,
		},
		{
			name:        "auto shows both the OCR and the description for short messages",
			command:     command{auto: true},
			altText:     []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			ocr:         []mediaResponse{{index: 0, responseType: foundOCRResponse, reply: message.Localized("my cool ocr")}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Localized("my cool description")}},
			expected:    "my cool description. It contains the text: my cool ocr",
		},
		{
			name:        "auto shows only the OCR for long messages",
			command:     command{auto: true},
			altText:     []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			ocr:         []mediaResponse{{index: 0, responseType: foundOCRResponse, reply: message.Localized(strings.Repeat("0123456789", 6))}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Localized("my cool description")}},
			expected:    strings.Repeat("0123456789", 6),
		},
		{
			name:        "auto shows the description if ocr fails and no alt text",
			command:     command{auto: true},
			altText:     []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			ocr:         []mediaResponse{{index: 0, responseType: foundOCRResponse, err: ocrErr}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Localized("my cool description")}},
			expected:    "my cool description",
		},
		{
			name:        "auto shows the description if no ocr or alt text",
			command:     command{auto: true},
			altText:     []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			ocr:         []mediaResponse{{index: 0, responseType: doNothingResponse}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, reply: message.Localized("my cool description")}},
			expected:    "my cool description",
		},
		{
			name:        "auto shows the description error if no ocr or alt text",
			command:     command{auto: true},
			altText:     []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			ocr:         []mediaResponse{{index: 0, responseType: doNothingResponse}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, err: describeErr}},
			expected:    "I'm at a loss for words, sorry!",
			hasErr:      true,
		},
		{
			name:        "auto prefers the OCR failure if everything blows up",
			command:     command{auto: true},
			altText:     []mediaResponse{{index: 0, responseType: missingAltTextResponse, reply: message.Localized(noAltText)}},
			ocr:         []mediaResponse{{index: 0, responseType: foundOCRResponse, err: ocrErr}},
			description: []mediaResponse{{index: 0, responseType: foundVisionResponse, err: describeErr}},
			expected:    "I'm at a loss for words, sorry!",
			hasErr:      true,
		},
		{
			name:        "auto tries to reply with a failure message if replying normally failed",
			command:     command{auto: true},
			altText:     []mediaResponse{{index: 0, responseType: foundAltTextResponse, reply: message.Localized(someText)}},
			ocr:         []mediaResponse{{index: 0, responseType: doNothingResponse}},
			description: []mediaResponse{{index: 0, responseType: doNothingResponse}},
			replyErr:    anErr,
			expected:    string(message.ErrorMessage(context.Background(), anErr)),
			hasErr:      true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer leaktest.Check(t)()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var origGetAltText = getAltText
			var origGetOcr = getOcr
			var origGetDescription = getDescription
			var origFindTweet = findTweet
			var origReply = _reply
			defer func() {
				getAltText = origGetAltText
				getOcr = origGetOcr
				getDescription = origGetDescription
				findTweet = origFindTweet
				_reply = origReply
			}()

			getAltText = func(ctx context.Context, mediaTweet *twitter.Tweet) []mediaResponse { return test.altText }
			getOcr = func(ctx context.Context, command command, mediaTweet *twitter.Tweet) []mediaResponse { return test.ocr }
			getDescription = func(ctx context.Context, mediaTweet *twitter.Tweet) []mediaResponse { return test.description }
			findTweet = func(ctx context.Context, client twitter.Twitter, tweet *twitter.Tweet) (*twitter.Tweet, structured_error.StructuredError) {
				// Golangs lack of generics are super-cool!
				numMedia := int(math.Max(math.Max(float64(len(test.altText)), float64(len(test.ocr))), float64(len(test.description))))
				media := make([]twitter.Media, numMedia)
				mediaTweet := &twitter.Tweet{
					Id: "mediaTweet",
					User: twitter.User{
						Id:       "",
						Username: "@someone",
						Display:  "UserPostingMedia",
					},
					Media: media,
				}
				return mediaTweet, test.findTweetErr
			}
			sentMessage := ""
			_reply = func(ctx context.Context, tweet *twitter.Tweet, message message.Localized) replier.ReplyResult {
				sentMessage = string(message)
				parentTweet := &twitter.Tweet{Id: "123"}
				return replier.ReplyResult{
					ParentTweet: parentTweet,
					Err:         test.replyErr,
				}
			}

			mockTwitter := &twitter_test.MockTwitter{T: t}

			ctx = WithHandleCommand(ctx, mockTwitter)
			ctx, err := replier.WithReplier(ctx, mockTwitter)
			assert.NoError(t, err)
			parentTweet := &twitter.Tweet{Id: "parentTweet"}
			result := handleCommand(ctx, test.command, parentTweet)
			if test.hasErr {
				require.Error(t, result.Err)
			} else {
				require.NoError(t, result.Err)
			}
			assert.Equal(t, test.expected, sentMessage)
		})
	}
}

func TestHandleCommandRespondsToAPanic(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var origReply = _reply
	defer func() {
		_reply = origReply
	}()

	didPanic := false
	sentMessage := ""
	_reply = func(ctx context.Context, tweet *twitter.Tweet, message message.Localized) replier.ReplyResult {
		// Panic the first time, but let the unknown reply go through
		if !didPanic {
			didPanic = true
			panic("touched the third rail")
		}
		sentMessage = string(message)
		parentTweet := &twitter.Tweet{Id: "123"}
		return replier.ReplyResult{
			ParentTweet: parentTweet,
			Err:         nil,
		}
	}

	mockTwitter := &twitter_test.MockTwitter{T: t}

	ctx = WithHandleCommand(ctx, mockTwitter)
	ctx, err := replier.WithReplier(ctx, mockTwitter)
	assert.NoError(t, err)
	assert.Panics(t, func() {
		HandleCommand(ctx, "help", &twitter.Tweet{})
	})
	unknownErr := structured_error.Wrap(errors.New("bad news bears"), structured_error.Unknown)
	expected := string(message.ErrorMessage(ctx, unknownErr))
	assert.Equal(t, expected, sentMessage)
}
