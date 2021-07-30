package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	vision "cloud.google.com/go/vision/apiv1"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

type google struct {
	client *vision.ImageAnnotatorClient
}

func NewGoogleVision(privateKeyId string, privateKey string) (OCR, error) {
	var ocr OCR
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
		var client *vision.ImageAnnotatorClient
		client, err = vision.NewImageAnnotatorClient(ctx, option.WithCredentialsJSON(credentialsJSON))
		if err == nil {
			ocr = &google{client: client}
		}
	}
	return ocr, err
}

func (g *google) GetOCR(url string) (*OCRResult, error) {
	var result *OCRResult
	image := vision.NewImageFromURI(url)
	ctx := context.Background()
	annotations, err := g.client.DetectDocumentText(ctx, image, nil)
	annotationsJSON, _ := json.Marshal(annotations)
	logrus.Debug(fmt.Sprintf("Google annotations\n%v\n", string(annotationsJSON)))
	if err == nil {
		text := getText(annotations.Pages)
		language := getLanguage(annotations.Pages)
		result = &OCRResult{Text: text, Language: language}
	}
	return result, err
}
func (g *google) Close() error {
	return g.client.Close()
}

func GoogleOCR(privateKeyId string, privateKey string, url string) (*pb.TextAnnotation, error) {
	credentials := map[string]string{
		"type":                        "service_account",
		"project_id":                  "captions-please-ocr",
		"private_key_id":              privateKeyId,
		"private_key":                 privateKey,
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
		var client *vision.ImageAnnotatorClient
		client, err = vision.NewImageAnnotatorClient(ctx, option.WithCredentialsJSON(credentialsJSON))
		if err == nil {
			defer client.Close()
			image := vision.NewImageFromURI(url)
			return client.DetectDocumentText(ctx, image, nil)
		}
	}
	return nil, err
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
