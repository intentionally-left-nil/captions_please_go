package handle_command

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/AnilRedshift/captions_please_go/pkg/vision"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
)

type ocrKey int

const theOcrKey ocrKey = 0

type ocrState struct {
	google     vision.OCR
	translator vision.Translator
}

type ocrJobResult struct {
	index int
	ocr   *vision.OCRResult
	err   structured_error.StructuredError
}

func WithOCR(ctx context.Context) (context.Context, error) {
	secrets := common.GetSecrets(ctx)
	google, err := vision.NewGoogle(secrets.GooglePrivateKeyID, secrets.GooglePrivateKeySecret)
	state := &ocrState{
		google:     google,
		translator: google.(vision.Translator),
	}

	go func() {
		<-ctx.Done()
		state.google.Close()
	}()
	return setOCRState(ctx, state), err
}

func getOCRMediaResponse(ctx context.Context, command command, mediaTweet *twitter.Tweet) []mediaResponse {
	state := getOCRState(ctx)
	wg := sync.WaitGroup{}
	wg.Add(len(mediaTweet.Media))

	jobs := make(chan ocrJobResult, len(mediaTweet.Media))
	for i, media := range mediaTweet.Media {
		i := i
		media := media
		go func() {
			var ocrResult *vision.OCRResult
			var err structured_error.StructuredError = nil
			if media.Type != "photo" {
				err = structured_error.Wrap(errors.New("media is not a photo"), structured_error.WrongMediaType)
			} else {
				ocrResult, err = state.google.GetOCR(ctx, media.Url)
				if err == nil && command.translate {
					shouldTranslate := ocrResult.Language.Confidence < 0.7
					if !shouldTranslate {
						desiredLanguage := message.GetLanguage(ctx)
						matcher := language.NewMatcher([]language.Tag{desiredLanguage})
						_, _, confidence := matcher.Match(ocrResult.Language.Tag)
						shouldTranslate = confidence < language.High
					}

					if shouldTranslate {
						translatedTag, translatedText, translatedErr := state.translator.Translate(ctx, ocrResult.Text)
						if translatedErr == nil {
							ocrResult = &vision.OCRResult{Text: translatedText, Language: vision.OCRLanguage{Tag: translatedTag, Confidence: 1.0}}
						} else {
							logrus.Error(fmt.Sprintf("Error %v trying to translate the OCR result", translatedErr))
						}
					}
				}

			}
			jobs <- ocrJobResult{index: i, ocr: ocrResult, err: err}
			wg.Done()
		}()
	}

	wg.Wait()
	close(jobs)
	jobResults := []ocrJobResult{}
	for job := range jobs {
		jobResults = append(jobResults, job)
	}
	sort.Slice(jobResults, func(i, j int) bool { return jobResults[i].index < jobResults[j].index })
	responses := make([]mediaResponse, len(mediaTweet.Media))

	for i := range mediaTweet.Media {
		var response mediaResponse
		jobResult := jobResults[i]
		if jobResult.err == nil {
			response = mediaResponse{index: i, responseType: foundOCRResponse, reply: message.Unlocalized(jobResult.ocr.Text)}
		} else if jobResult.err.Type() == structured_error.WrongMediaType {
			response = mediaResponse{index: i, responseType: doNothingResponse}
		} else {
			response = mediaResponse{index: i, responseType: foundOCRResponse, err: jobResult.err}
		}
		responses[i] = response
	}
	return responses
}

func setOCRState(ctx context.Context, state *ocrState) context.Context {
	return context.WithValue(ctx, theOcrKey, state)
}

func getOCRState(ctx context.Context) *ocrState {
	return ctx.Value(theOcrKey).(*ocrState)
}
