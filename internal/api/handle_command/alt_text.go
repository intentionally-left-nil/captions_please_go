package handle_command

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/AnilRedshift/captions_please_go/pkg/vision"
	"github.com/sirupsen/logrus"
)

type altTextCtxKey int

const theAltTextCtxKey altTextCtxKey = 0

type altTextState struct {
	translator vision.Translator
}

func WithAltText(ctx context.Context) (context.Context, error) {
	secrets := common.GetSecrets(ctx)
	translator, err := vision.NewGoogle(secrets.GooglePrivateKeyID, secrets.GooglePrivateKeySecret)
	if err == nil {
		state := &altTextState{translator: translator}
		ctx = context.WithValue(ctx, theAltTextCtxKey, state)
		go func() {
			<-ctx.Done()
			state.translator.Close()
		}()
	}
	return ctx, err
}

func getAltTextMediaResponse(ctx context.Context, command command, mediaTweet *twitter.Tweet) []mediaResponse {
	state := ctx.Value(theAltTextCtxKey).(*altTextState)
	wg := sync.WaitGroup{}
	wg.Add(len(mediaTweet.Media))

	jobs := make(chan mediaResponse, len(mediaTweet.Media))
	for i, media := range mediaTweet.Media {
		go func(i int, media twitter.Media) {
			defer wg.Done()
			var response mediaResponse
			if media.AltText != nil {
				response = mediaResponse{index: i, responseType: foundAltTextResponse, reply: message.Localized(*media.AltText)}
			} else if media.Type == "photo" {
				reply := message.NoAltText(ctx, mediaTweet.User.Display)
				response = mediaResponse{index: i, responseType: missingAltTextResponse, reply: reply}
			} else {
				response = mediaResponse{index: i, responseType: doNothingResponse}
			}

			if command.translate && response.responseType == foundAltTextResponse {
				_, translation, translateErr := state.translator.Translate(ctx, string(response.reply))
				if translateErr == nil {
					response.reply = message.Localized(translation)
				} else {
					logrus.Error(fmt.Sprintf("Alt text encountered an error %v when translating", translateErr))
				}
			}
			jobs <- response

		}(i, media)
	}
	wg.Wait()
	close(jobs)

	responses := []mediaResponse{}
	for response := range jobs {
		responses = append(responses, response)
	}
	sort.Slice(responses, func(i, j int) bool { return responses[i].index < responses[j].index })
	return responses
}
