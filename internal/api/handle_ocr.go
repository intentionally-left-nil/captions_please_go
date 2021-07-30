package api

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/AnilRedshift/captions_please_go/pkg/vision"
	"github.com/sirupsen/logrus"
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
	return context.WithValue(ctx, theOcrKey, state), err
}

func HandleOCR(ctx context.Context, tweet *twitter.Tweet) <-chan ActivityResult {
	state := getOCRState(ctx)
	mediaTweet, err := findTweetWithMedia(ctx, state.client, tweet)
	var ErrNoPhotosFoundType *ErrNoPhotosFound
	var ErrWrongMediaTypeType *ErrWrongMediaType
	reply := ""
	if err == nil {
		photos := getPhotos(mediaTweet.Media)
		wg := sync.WaitGroup{}
		wg.Add(len(photos))

		photoJobs := make(chan ocrJobResult, len(photos))
		for i, photo := range photos {
			i := i
			photo := photo
			go func() {
				ocrResult, err := state.google.GetOCR(photo.Url)
				photoJobs <- ocrJobResult{index: i, ocr: ocrResult, err: err}
				wg.Done()
			}()
		}

		wg.Wait()
		close(photoJobs)
		results := []ocrJobResult{}
		for job := range photoJobs {
			results = append(results, job)
		}
		sort.Slice(results, func(i, j int) bool { return results[i].index < results[j].index })
		messages := []string{}
		for _, result := range results {
			if result.err == nil {
				messages = append(messages, result.ocr.Text)
			} else {
				message := "I encountered difficulties scanning the message. Sorry!"
				messages = append(messages, message)
			}
		}

		if len(messages) > 1 {
			for i, message := range messages {
				messages[i] = fmt.Sprintf("Image %d: %s", i+1, message)
			}
		}
		reply = strings.Join(messages, "\n")
	} else if errors.As(err, &ErrNoPhotosFoundType) {
		reply = "I didn't find any photos to interpret, but I appreciate the shoutout!. Try \"@captions_please help\" to learn more"
	} else if errors.As(err, &ErrWrongMediaTypeType) {
		reply = "I only know how to interpret photos right now, sorry!"
	} else {
		reply = "My joints are freezing up! Hey @TheOtherAnil can you please fix me?"
	}
	// Even if there's an error we want to try and send a response
	_, sendErr := replyWithMultipleTweets(ctx, state.client, tweet.Id, reply)

	if sendErr != nil {
		logrus.Info(fmt.Sprintf("Failed to send response %s to tweet %s with error %v", reply, tweet.Id, sendErr))

		if err == nil {
			err = sendErr
		}
	}
	out := make(chan ActivityResult, 1)
	out <- ActivityResult{tweet: tweet, err: err, action: "reply with ocr"}
	close(out)
	return out
}

func getOCRState(ctx context.Context) *ocrState {
	return ctx.Value(theOcrKey).(*ocrState)
}
