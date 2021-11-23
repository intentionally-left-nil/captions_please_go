package vision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/storage"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"

	"cloud.google.com/go/translate"
	vision "cloud.google.com/go/vision/apiv1"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

type google struct {
	visionClient     *vision.ImageAnnotatorClient
	translateClient  *translate.Client
	transcribeClient *speech.Client
	storageClient    *storage.Client
	supportedTags    []language.Tag
}

type Google interface {
	OCR
	Translator
	Transcriber
}

func NewGoogle(privateKeyId string, privateKey string) (Google, error) {
	var Google Google
	credentials := map[string]string{
		"type":                        "service_account",
		"project_id":                  "captions-please-ocr",
		"private_key_id":              privateKeyId,
		"private_key":                 strings.ReplaceAll(privateKey, "\\n", "\n"),
		"client_email":                "captions-please@captions-please-ocr.iam.gserviceaccount.com",
		"client_id":                   "101126542430586578005",
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/captions-please%40captions-please-ocr.iam.gserviceaccount.com",
	}

	credentialsJSON, err := json.Marshal(credentials)
	if err == nil {
		ctx := context.Background()
		var visionClient *vision.ImageAnnotatorClient
		var translateClient *translate.Client
		var transcribeClient *speech.Client
		var storageClient *storage.Client
		visionClient, err = vision.NewImageAnnotatorClient(ctx, option.WithCredentialsJSON(credentialsJSON))
		if err == nil {
			translateClient, err = translate.NewClient(ctx, option.WithCredentialsJSON(credentialsJSON))
			if err == nil {
				transcribeClient, err = speech.NewClient(ctx, option.WithCredentialsJSON(credentialsJSON))
				if err == nil {
					storageClient, err = storage.NewClient(ctx, option.WithCredentialsJSON(credentialsJSON))
				}
				Google = &google{
					visionClient:     visionClient,
					translateClient:  translateClient,
					transcribeClient: transcribeClient,
					storageClient:    storageClient,
				}
			}
		}
	}
	return Google, err
}

func (g *google) GetOCR(ctx context.Context, url string) (*OCRResult, structured_error.StructuredError) {
	var result *OCRResult
	image := vision.NewImageFromURI(url)
	annotations, err := g.visionClient.DetectDocumentText(ctx, image, nil)
	if annotations == nil && err == nil {
		err = errors.New("no results")
	}
	// This is too much, even for debug spew
	// annotationsJSON, _ := json.Marshal(annotations)
	// logrus.Debug(fmt.Sprintf("Google annotations\n%v\nerr: %v", string(annotationsJSON), err))
	logrus.Debug(fmt.Sprintf("Have Google annotations %v", annotations != nil))
	if err == nil {
		text := getText(annotations.Pages)
		language := getLanguage(annotations.Pages)
		result = &OCRResult{Text: text, Language: language}
	}
	return result, structured_error.Wrap(err, structured_error.OCRError)
}

func (g *google) Close() error {
	errs := []error{
		g.visionClient.Close(),
		g.translateClient.Close(),
		g.transcribeClient.Close(),
		g.storageClient.Close(),
	}
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *google) Translate(ctx context.Context, toTranslate string) (language.Tag, string, structured_error.StructuredError) {
	var tag language.Tag
	var translated string
	var err error
	g.loadSupportedLanguages(ctx)
	if len(g.supportedTags) == 0 {
		err = errors.New("cannot determine supported tags for translation")
	}
	if err == nil {
		tag, err = message.GetCompatibleLanguage(ctx, g.supportedTags)
		if err == nil {
			var translations []translate.Translation
			logrus.Debug(fmt.Sprintf("Calling translate with tag %s", tag.String()))
			translations, err = g.translateClient.Translate(ctx, []string{toTranslate}, tag, &translate.Options{
				Format: translate.Text,
			})
			if len(translations) == 0 && err == nil {
				err = errors.New("no results")
			}

			if err == nil {
				texts := make([]string, len(translations))
				for i, translation := range translations {
					texts[i] = translation.Text
				}
				translated = strings.Join(texts, "\n")
				logrus.Debug(fmt.Sprintf("successfully translated %s into %s for language %s", toTranslate, translated, tag.String()))
			}
		}
	}

	if err != nil {
		logrus.Debug(fmt.Sprintf("Translation failed with %v", err))
	}
	return tag, translated, structured_error.Wrap(err, structured_error.TranslateError)
}

func (g *google) Transcribe(ctx context.Context, url string) ([]TranscriptionResult, structured_error.StructuredError) {
	var results []TranscriptionResult
	objectName := uuid.New().String()
	err := g.withAudioFromVideoURL(ctx, url, func(filename string) error {
		return g.uploadFile(ctx, filename, objectName)
	})
	if err == nil {
		defer g.getCloudObject(objectName).Delete(ctx)
		language := message.GetLanguage(ctx)
		var operation *speech.LongRunningRecognizeOperation

		operation, err = g.transcribeClient.LongRunningRecognize(ctx, &speechpb.LongRunningRecognizeRequest{
			Config: &speechpb.RecognitionConfig{
				Encoding:                            speechpb.RecognitionConfig_FLAC,
				EnableSeparateRecognitionPerChannel: false,
				LanguageCode:                        language.String(),
				MaxAlternatives:                     1,
				ProfanityFilter:                     false,
				SpeechContexts:                      []*speechpb.SpeechContext{},
				EnableWordTimeOffsets:               false,
				EnableAutomaticPunctuation:          false,
				DiarizationConfig:                   &speechpb.SpeakerDiarizationConfig{},
				Metadata:                            &speechpb.RecognitionMetadata{},
				Model:                               "video",
				UseEnhanced:                         false,
			},
			Audio: &speechpb.RecognitionAudio{
				AudioSource: &speechpb.RecognitionAudio_Uri{
					Uri: fmt.Sprintf("gs://captions_please_transcribe/%s", objectName),
				},
			},
		})

		if err == nil {
			var response *speechpb.LongRunningRecognizeResponse
			response, err = operation.Wait(ctx)
			if err == nil {
				results = []TranscriptionResult{}
				if err == nil && response != nil && response.Results != nil {
					for _, result := range response.Results {
						if result != nil && len(result.Alternatives) > 0 && result.Alternatives[0].Transcript != "" {
							results = append(results, TranscriptionResult{Confidence: result.Alternatives[0].Confidence, Text: result.Alternatives[0].Transcript})
						}
					}
				}

				if len(results) == 0 && err == nil {
					err = errors.New("no results")
				}
			}
		}
	}

	if err != nil {
		logrus.Debug(fmt.Sprintf("Transcription failed with %v", err))
	}
	return results, structured_error.Wrap(err, structured_error.TranscribeError)
}

func (g *google) withAudioFromVideoURL(ctx context.Context, url string, handler func(filename string) error) error {
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err == nil {
		var audioResp *http.Response
		audioResp, err = http.DefaultClient.Do(httpRequest)
		if err == nil && audioResp.StatusCode < 200 && audioResp.StatusCode >= 300 {
			err = fmt.Errorf("downloading the URL failed with status code %d", audioResp.StatusCode)
		}
		if err == nil {
			defer audioResp.Body.Close()
			var dir string
			dir, err = ioutil.TempDir("", "captions_please_transcribe")
			if err == nil {
				defer os.RemoveAll(dir)
				sourceName := filepath.Join(dir, "source.mp4")
				destName := filepath.Join(dir, "out.flac")
				var source *os.File
				source, err = os.Create(sourceName)
				if err == nil {
					_, err = io.Copy(source, audioResp.Body)
					if err == nil {
						err = exec.Command("ffmpeg", "-i", sourceName, "-vn", "-c:v", "flac", destName).Run()
						if err == nil {
							err = handler(destName)
						}
					}
				}
			}
		}
	}
	return err
}

func (g *google) uploadFile(ctx context.Context, filename string, objectName string) error {
	file, err := os.Open(filename)
	if err == nil {
		defer file.Close()
		writer := g.getCloudObject(objectName).NewWriter(ctx)
		_, err = io.Copy(writer, file)
		if err == nil {
			err = writer.Close()
		}
	}
	return err
}

func (g *google) getCloudObject(objectName string) *storage.ObjectHandle {
	return g.storageClient.Bucket("captions_please_transcribe").Object(objectName)
}

func (g *google) loadSupportedLanguages(ctx context.Context) {
	if len(g.supportedTags) == 0 {
		languages, err := g.translateClient.SupportedLanguages(ctx, language.English)
		if err == nil {
			tags := make([]language.Tag, len(languages))
			for i, lang := range languages {
				tags[i] = lang.Tag
			}
			g.supportedTags = tags
		}
	}
}

func getLanguage(pages []*pb.Page) OCRLanguage {
	numPages := len(pages)
	if numPages == 0 {
		return OCRLanguage{}
	}

	languages := map[string][]float32{}
	for _, page := range pages {
		// Apparently, google returns nil pointers sometimes so we have to check everything. sigh.
		if page.Property != nil {
			for _, detectedLanguage := range page.Property.DetectedLanguages {
				confidences, ok := languages[detectedLanguage.LanguageCode]
				if !ok {
					confidences = []float32{}
				}
				languages[detectedLanguage.LanguageCode] = append(confidences, float32(detectedLanguage.Confidence))
			}
		}
	}

	logrus.Debug(fmt.Sprintf("Languages %v\n", languages))

	ocrLanguage := OCRLanguage{Tag: language.English, Confidence: 0.0}
	for code, confidences := range languages {
		tag, err := language.Parse(code)
		if err == nil {

			var confidence float32 = 0.0
			for _, val := range confidences {
				confidence += val
			}
			confidence = confidence / float32(numPages)
			if confidence > ocrLanguage.Confidence {
				ocrLanguage.Tag = tag
				ocrLanguage.Confidence = confidence
			}
		}
	}
	return ocrLanguage
}

func getText(pages []*pb.Page) string {
	builder := strings.Builder{}
	// Apparently, google returns nil pointers sometimes so we have to check everything. sigh.
	for _, page := range pages {
		if page == nil {
			continue
		}
		for _, block := range page.Blocks {
			if block == nil {
				continue
			}
			for _, paragraph := range block.Paragraphs {
				if paragraph == nil {
					continue
				}
				for _, word := range paragraph.Words {
					if word == nil {
						continue
					}
					for _, symbol := range word.Symbols {
						if symbol == nil {
							continue
						}
						if symbol.Property != nil && symbol.Property.DetectedBreak != nil && symbol.Property.DetectedBreak.IsPrefix {
							builder.WriteString(" ")
						}
						builder.WriteString(symbol.Text)
						if symbol.Property != nil && symbol.Property.DetectedBreak != nil && !symbol.Property.DetectedBreak.IsPrefix {
							builder.WriteString(" ")
						}
					}
				}
				builder.WriteString("\n\n")
			}
		}
	}
	text := strings.TrimSpace(builder.String())
	return text
}
