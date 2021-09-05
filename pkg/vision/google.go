package vision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/translate"
	vision "cloud.google.com/go/vision/apiv1"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
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
	visionErr := g.visionClient.Close()
	translateErr := g.translateClient.Close()
	if visionErr != nil {
		return visionErr
	}
	return translateErr
}

func (g *google) Translate(ctx context.Context, toTranslate string) (language.Tag, string, structured_error.StructuredError) {
	var tag language.Tag
	var translated string
	var err error
	g.loadSupportedLanguages(ctx)
	if g.matcher != nil {
		tag, err = message.GetCompatibleLanguage(ctx, *g.matcher)
		if err == nil {
			var translations []translate.Translation
			translations, err = g.translateClient.Translate(ctx, []string{toTranslate}, tag, &translate.Options{
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
			logrus.Debug(fmt.Sprintf("successfully translated %s into %s", toTranslate, translated))
		}
	}

	if err != nil {
		logrus.Debug(fmt.Sprintf("Translation failed with %v", err))
	}
	return tag, translated, structured_error.Wrap(err, structured_error.TranslateError)
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
