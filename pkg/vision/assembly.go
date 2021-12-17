package vision

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/sirupsen/logrus"
)

type assemblyAiTransport struct {
	http.RoundTripper
	key string
}

func (t *assemblyAiTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request.Header.Set("Authorization", t.key)
	request.Header.Set("Content-Type", "application/json")
	return http.DefaultTransport.RoundTrip(request)
}

type assemblyAi struct {
	client *http.Client
}

type AssemblyAi interface {
	Transcriber
}

func NewAssemblyAi(key string) AssemblyAi {
	return &assemblyAi{
		client: &http.Client{
			Transport: &assemblyAiTransport{key: key},
		},
	}
}

func (a *assemblyAi) Transcribe(ctx context.Context, url string) ([]TranscriptionResult, structured_error.StructuredError) {
	var result []TranscriptionResult
	type transcriptRequest struct {
		AudioUrl string `json:"audio_url"`
	}
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(transcriptRequest{AudioUrl: url})
	if err != nil {
		return nil, structured_error.Wrap(err, structured_error.TranscribeError)
	}
	request, err := http.NewRequest(http.MethodPost, "https://api.assemblyai.com/v2/transcript", buf)
	ctx, onComplete := context.WithTimeout(ctx, time.Second*30)
loop:
	for {
		if err != nil {
			break loop
		}
		var response *http.Response
		response, err = a.client.Do(request)
		select {
		case <-ctx.Done():
			err = errors.New("timeout")
		default:
			if err == nil && (response.StatusCode < 200 || response.StatusCode >= 300) {
				err = fmt.Errorf("transcript returned a %d status code", response.StatusCode)
			}
			if err == nil {
				defer response.Body.Close()
				type transcriptResponse struct {
					Id         string  `json:"id"`
					Status     string  `json:"status"`
					Confidence float32 `json:"confidence"`
					Text       string  `json:"text"`
					Error      string  `json:"error"`
				}
				var parsed transcriptResponse
				err = json.NewDecoder(response.Body).Decode(&parsed)
				if err == nil {
					logrus.Debug(fmt.Sprintf("AssemblyAI returned %v", parsed))
					if parsed.Status == "completed" {
						if len(parsed.Text) == 0 {
							err = errors.New("no results")
						} else {
							result = []TranscriptionResult{{Confidence: parsed.Confidence, Text: parsed.Text}}
						}
						break loop
					} else if parsed.Status == "error" {
						err = errors.New(parsed.Error)
					} else {
						request, err = http.NewRequest(http.MethodGet, "https://api.assemblyai.com/v2/transcript/"+parsed.Id, nil)
					}
				}
			}
		}
		time.Sleep(time.Millisecond * 100)
	}
	onComplete()
	return result, structured_error.Wrap(err, structured_error.TranscribeError)
}
