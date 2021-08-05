package vision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/translate"
	vision "cloud.google.com/go/vision/apiv1"
	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

type google struct {
	visionClient    *vision.ImageAnnotatorClient
	translateClient *translate.Client
	matcher         *language.Matcher
}

type Google interface {
	OCR
	Translator
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
		visionClient, err = vision.NewImageAnnotatorClient(ctx, option.WithCredentialsJSON(credentialsJSON))
		if err == nil {
			translateClient, err = translate.NewClient(ctx, option.WithCredentialsJSON(credentialsJSON))
			if err == nil {
				Google = &google{visionClient: visionClient, translateClient: translateClient}
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
	annotationsJSON, _ := json.Marshal(annotations)
	logrus.Debug(fmt.Sprintf("Google annotations\n%v\nerr: %v", string(annotationsJSON), err))
	if err == nil {
		text := getText(annotations.Pages)
		language := getLanguage(annotations.Pages)
		result = &OCRResult{Text: text, Language: language}
	}
	return result, structured_error.Wrap(err, structured_error.OCRError)
}

func (g *google) Close() error {
	visionErr := g.visionClient.Close()
	translateErr := g.translateClient.Close()
	if visionErr != nil {
		return visionErr
	}
	return translateErr
}

func (g *google) Translate(ctx context.Context, message string) (string, structured_error.StructuredError) {
	var translated string
	var err error
	g.loadSupportedLanguages(ctx)
	if g.matcher != nil {
		desired := replier.GetLanguage(ctx)
		tag, _, confidence := (*g.matcher).Match(desired)
		if confidence >= language.High {
			var translations []translate.Translation
			translations, err = g.translateClient.Translate(ctx, []string{message}, tag, &translate.Options{
				Format: translate.Text,
			})
			if len(translations) == 0 && err == nil {
				err = errors.New("no results")
			}
			texts := make([]string, len(translations))
			for i, translation := range translations {
				texts[i] = translation.Text
			}
			translated = strings.Join(texts, "\n")
		} else {
			err = structured_error.Wrap(fmt.Errorf("%v has no high confidence matching language", desired), structured_error.UnsupportedLanguage)
		}
	}
	return translated, structured_error.Wrap(err, structured_error.TranslateError)
}

func (g *google) loadSupportedLanguages(ctx context.Context) {
	if g.matcher == nil {
		languages, err := g.translateClient.SupportedLanguages(ctx, language.English)
		if err == nil {
			tags := make([]language.Tag, len(languages))
			for i, lang := range languages {
				tags[i] = lang.Tag
			}
			matcher := language.NewMatcher(tags)
			g.matcher = &matcher
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
		for _, detectedLanguage := range page.Property.DetectedLanguages {
			confidences, ok := languages[detectedLanguage.LanguageCode]
			if !ok {
				confidences = []float32{}
			}
			languages[detectedLanguage.LanguageCode] = append(confidences, float32(detectedLanguage.Confidence))
		}
	}

	logrus.Debug(fmt.Sprintf("Languages %v\n", languages))

	language := OCRLanguage{}
	for code, confidences := range languages {
		var confidence float32 = 0.0
		for _, val := range confidences {
			confidence += val
		}
		confidence = confidence / float32(numPages)
		if confidence > language.Confidence {
			language.Code = code
			language.Confidence = confidence
		}

	}
	return language
}

func getText(pages []*pb.Page) string {
	builder := strings.Builder{}
	for _, page := range pages {
		for _, block := range page.Blocks {
			for _, paragraph := range block.Paragraphs {
				for _, word := range paragraph.Words {
					for _, symbol := range word.Symbols {
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
