package handle_command

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/AnilRedshift/captions_please_go/pkg/vision"
)

type ocrKey int

const theOcrKey ocrKey = 0

type ocrState struct {
	google vision.OCR
	client twitter.Twitter
}

type ocrJobResult struct {
	index int
	ocr   *vision.OCRResult
	err   structured_error.StructuredError
}

func WithOCR(ctx context.Context, client twitter.Twitter) (context.Context, error) {
	secrets := common.GetSecrets(ctx)
	google, err := vision.NewGoogleVision(secrets.GooglePrivateKeyID, secrets.GooglePrivateKeySecret)
	state := &ocrState{
		google: google,
		client: client,
	}

	go func() {
		<-ctx.Done()
		state.google.Close()
	}()
	return setOCRState(ctx, state), err
}

func HandleOCR(ctx context.Context, tweet *twitter.Tweet) common.ActivityResult {
	state := getOCRState(ctx)
	response := combineAndSendResponses(ctx, state.client, tweet, getOCRMediaResponse)
	response.Action = "reply with OCR"
	return response
}

func getOCRMediaResponse(ctx context.Context, tweet *twitter.Tweet, mediaTweet *twitter.Tweet) []mediaResponse {
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
			response = mediaResponse{index: i, responseType: foundOCRResponse, reply: replier.Unlocalized(jobResult.ocr.Text)}
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
