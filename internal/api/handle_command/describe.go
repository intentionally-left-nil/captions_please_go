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
)

type describeKey int

const theDescribeKey describeKey = 0

type describeState struct {
	describer  vision.Describer
	translator vision.Translator
}

type visionJobResult struct {
	index   int
	results []vision.VisionResult
	err     structured_error.StructuredError
}

const lowVisionConfidenceCutoff = 0.25

func WithDescribe(ctx context.Context) (context.Context, error) {
	secrets := common.GetSecrets(ctx)
	translator, err := vision.NewGoogle(secrets.GooglePrivateKeyID, secrets.GooglePrivateKeySecret)
	if err != nil {
		return ctx, err
	}
	describer := vision.NewAzureVision(secrets.AzureComputerVisionKey)
	state := describeState{
		describer:  describer,
		translator: translator,
	}
	go func() {
		<-ctx.Done()
		state.translator.Close()
	}()
	return setDescribeState(ctx, &state), err
}

func setDescribeState(ctx context.Context, state *describeState) context.Context {
	return context.WithValue(ctx, theDescribeKey, state)
}

func getDescriberState(ctx context.Context) *describeState {
	return ctx.Value(theDescribeKey).(*describeState)
}

func getDescribeMediaResponse(ctx context.Context, mediaTweet *twitter.Tweet) []mediaResponse {
	state := getDescriberState(ctx)

	wg := sync.WaitGroup{}
	wg.Add(len(mediaTweet.Media))
	jobs := make(chan visionJobResult, len(mediaTweet.Media))
	for i, media := range mediaTweet.Media {
		i := i
		media := media
		go func() {
			if media.Type == "photo" {
				visionResult, err := state.describer.Describe(ctx, media.Url)
				if err != nil && err.Type() == structured_error.UnsupportedLanguage {
					logrus.Debug("The results are valid, but in the wrong language. Trying to translate")
					translatedResult := make([]vision.VisionResult, len(visionResult))
					for i, result := range visionResult {
						var translated string
						translated, err = state.translator.Translate(ctx, result.Text)
						if err != nil {
							break
						}
						result.Text = translated
						// range() makes a copy, so we need to store the modifications
						translatedResult[i] = result
					}

					if err == nil {
						visionResult = translatedResult
					}
				}
				jobs <- visionJobResult{index: i, results: visionResult, err: err}
			} else {
				jobs <- visionJobResult{index: i, err: structured_error.Wrap(errors.New("media is not a photo"), structured_error.WrongMediaType)}
			}
			wg.Done()
		}()
	}

	wg.Wait()
	close(jobs)
	jobResults := []visionJobResult{}
	for job := range jobs {
		jobResults = append(jobResults, job)
	}
	sort.Slice(jobResults, func(i, j int) bool { return jobResults[i].index < jobResults[j].index })
	responses := make([]mediaResponse, len(mediaTweet.Media))
	for i := range mediaTweet.Media {
		var response mediaResponse
		jobResult := jobResults[i]
		if jobResult.err == nil {
			reply, err := formatVisionReply(ctx, jobResult.results)
			response = mediaResponse{index: i, responseType: foundVisionResponse, reply: reply, err: err}
		} else if jobResult.err.Type() == structured_error.WrongMediaType {
			response = mediaResponse{index: i, responseType: doNothingResponse}
		} else {
			response = mediaResponse{index: i, responseType: foundVisionResponse, err: jobResult.err}

		}
		responses[i] = response
	}
	return responses
}

func formatVisionReply(ctx context.Context, visionResults []vision.VisionResult) (message.Localized, structured_error.StructuredError) {
	var localized message.Localized
	var err structured_error.StructuredError = nil
	filteredResults := make([]string, 0, len(visionResults))
	for i, visionResult := range visionResults {
		if i > 2 || visionResult.Confidence < lowVisionConfidenceCutoff {
			break
		}
		filteredResults = append(filteredResults, visionResult.Text)
	}

	if len(filteredResults) == 0 {
		err = structured_error.Wrap(fmt.Errorf("there were %d results, but none were high-confidence", len(visionResults)), structured_error.DescribeError)
	} else {
		localized = message.CombineDescriptions(ctx, filteredResults)
	}
	return localized, err
}
