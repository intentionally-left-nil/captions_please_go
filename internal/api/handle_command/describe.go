package handle_command

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/AnilRedshift/captions_please_go/pkg/vision"
)

type describeKey int

const theDescribeKey describeKey = 0

type describeState struct {
	describer vision.Describer
	client    twitter.Twitter
}

type visionJobResult struct {
	index   int
	results []vision.VisionResult
	err     error
}

const lowVisionConfidenceCutoff = 0.25

func WithDescribe(ctx context.Context, client twitter.Twitter) context.Context {
	secrets := common.GetSecrets(ctx)
	describer := vision.NewAzureVision(secrets.AzureComputerVisionKey)
	state := describeState{
		describer: describer,
		client:    client,
	}
	return setDescribeState(ctx, &state)
}

func setDescribeState(ctx context.Context, state *describeState) context.Context {
	return context.WithValue(ctx, theDescribeKey, state)
}

func getDescriberState(ctx context.Context) *describeState {
	return ctx.Value(theDescribeKey).(*describeState)
}

func HandleDescribe(ctx context.Context, tweet *twitter.Tweet) <-chan common.ActivityResult {
	state := getDescriberState(ctx)
	var err error
	mediaTweet, err := findTweetWithMedia(ctx, state.client, tweet)
	if err == nil {
		responses := getDescribeMediaResponse(ctx, tweet, mediaTweet)
		responses = removeDoNothings(responses)

		replies := extractReplies(responses, func(response mediaResponse) string {
			err = response.err
			reply := response.reply
			if reply == "" {
				reply = "I encountered difficulties interpreting the image. Sorry!"
			}
			return reply
		})
		sendErr := sendReplies(ctx, state.client, tweet, replies)
		if err == nil {
			err = sendErr
		}
	} else {
		sendReplyForBadMedia(ctx, state.client, tweet, err)
	}
	out := make(chan common.ActivityResult, 1)
	out <- common.ActivityResult{Tweet: tweet, Err: err, Action: "reply with description"}
	close(out)
	return out
}

func getDescribeMediaResponse(ctx context.Context, tweet *twitter.Tweet, mediaTweet *twitter.Tweet) []mediaResponse {
	state := getDescriberState(ctx)

	wg := sync.WaitGroup{}
	wg.Add(len(mediaTweet.Media))
	jobs := make(chan visionJobResult, len(mediaTweet.Media))
	for i, media := range mediaTweet.Media {
		i := i
		media := media
		go func() {
			if media.Type == "photo" {
				visionResult, err := state.describer.Describe(media.Url)
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
			reply, err := formatVisionReply(jobResult.results)
			response = mediaResponse{index: i, responseType: foundVisionResponse, reply: reply, err: err}
		} else {
			// TODO remove wrapping once this is converted to a structured error
			sErr := structured_error.Wrap(jobResult.err, structured_error.Unknown)
			if sErr.Type() == structured_error.WrongMediaType {
				response = mediaResponse{index: i, responseType: doNothingResponse}
			} else {
				response = mediaResponse{index: i, responseType: foundVisionResponse, err: jobResult.err}

			}
		}

		responses[i] = response
	}
	return responses
}

func formatVisionReply(visionResults []vision.VisionResult) (string, structured_error.StructuredError) {
	var err structured_error.StructuredError = nil
	filteredResults := make([]vision.VisionResult, 0, len(visionResults))
	for i, visionResult := range visionResults {
		if i > 2 || visionResult.Confidence < lowVisionConfidenceCutoff {
			break
		}
		filteredResults = append(filteredResults, visionResult)
	}

	reply := ""
	if len(filteredResults) == 0 {
		reply = "I'm at a loss for words, sorry!"
		err = structured_error.Wrap(fmt.Errorf("there were %d results, but none were high-confidence", len(visionResults)), structured_error.NoHighConfidenceResults)
	} else {
		reply = filteredResults[0].Text
		for _, result := range filteredResults[1:] {
			reply = reply + ". It might also be " + result.Text
		}
	}
	return reply, err
}
