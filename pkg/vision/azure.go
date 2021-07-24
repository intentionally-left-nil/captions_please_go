package vision

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/cognitiveservices/v3.1/computervision"
	"github.com/Azure/go-autorest/autorest"
)

type azure struct {
	client computervision.BaseClient
}

func NewAzureVision(computerVisionKey string) Describer {
	client := computervision.New("https://captionspleasecomputervision.cognitiveservices.azure.com")
	client.Authorizer = autorest.NewCognitiveServicesAuthorizer(computerVisionKey)
	return &azure{client: client}
}

const captionThreshold = 0.4

func (a *azure) Describe(url string) ([]VisionResult, error) {
	var result []VisionResult
	ctx := context.Background()
	imageURL := computervision.ImageURL{URL: &url}
	description, err := a.client.DescribeImage(ctx, imageURL, nil, "en", nil)
	if err == nil && description.Captions != nil {
		result = make([]VisionResult, 0, len(*description.Captions))
		for i, caption := range *description.Captions {
			if caption.Confidence != nil && caption.Text != nil && *caption.Confidence > captionThreshold {
				result = result[:len(result)+1]
				result[i] = VisionResult{Text: *caption.Text, Confidence: float32(*caption.Confidence)}
			}
		}
	}
	return result, err
}

func (a *azure) GetOCR(url string) (*OCRResult, error) {
	var ocr *OCRResult
	ctx := context.Background()
	imageURL := computervision.ImageURL{URL: &url}
	result, err := a.client.RecognizePrintedText(ctx, true, imageURL, computervision.OcrLanguagesUnk)
	builder := strings.Builder{}
	if err == nil && result.Regions != nil {
		for _, region := range *result.Regions {
			for _, line := range *region.Lines {
				for _, word := range *line.Words {
					builder.WriteString(*word.Text + " ")
				}
				builder.WriteString(" ")
			}
			builder.WriteString("\n\n")
		}
		language := ""
		if result.Language != nil {
			language = *result.Language
		}
		ocr = &OCRResult{
			Text:     builder.String(),
			Language: OCRLanguage{Code: language, Confidence: 1.0},
		}
	}
	return ocr, err
}
