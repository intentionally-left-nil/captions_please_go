package api

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
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
	err   error
}

func WithOCR(ctx context.Context, client twitter.Twitter) (context.Context, error) {
	secrets := GetSecrets(ctx)
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

func HandleOCR(ctx context.Context, tweet *twitter.Tweet) <-chan common.ActivityResult {
	state := getOCRState(ctx)
	mediaTweet, err := findTweetWithMedia(ctx, state.client, tweet)
	if err == nil {
		responses := getOCRMediaResponse(ctx, tweet, mediaTweet)
		responses = removeDoNothings(responses)

		replies := extractReplies(responses, func(response mediaResponse) string {
			err = response.err
			return "I encountered difficulties scanning the image. Sorry!"
		})
		sendErr := sendReplies(ctx, state.client, tweet, replies)
		if err == nil {
			err = sendErr
		}
	} else {
		sendReplyForBadMedia(ctx, state.client, tweet, err)
	}
	out := make(chan common.ActivityResult, 1)
	out <- common.ActivityResult{Tweet: tweet, Err: err, Action: "reply with OCR"}
	close(out)
	return out
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
			if media.Type == "photo" {
				ocrResult, err := state.google.GetOCR(media.Url)
				jobs <- ocrJobResult{index: i, ocr: ocrResult, err: err}
			} else {
				jobs <- ocrJobResult{index: i, err: &ErrWrongMediaType{}}
			}
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

	var ErrWrongMediaTypeType *ErrWrongMediaType
	for i := range mediaTweet.Media {
		var response mediaResponse
		jobResult := jobResults[i]
		if jobResult.err == nil {
			response = mediaResponse{index: i, responseType: foundOCRResponse, reply: jobResult.ocr.Text}
		} else if errors.As(jobResult.err, &ErrWrongMediaTypeType) {
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
