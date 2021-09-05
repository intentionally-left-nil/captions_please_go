package vision

import (
	"context"
	"encoding/json"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
)

type OCRLanguage struct {
	Tag        language.Tag
	Confidence float32
}

type OCRResult struct {
	Text     string
	Language OCRLanguage
}

type VisionResult struct {
	Text       string
	Confidence float32
}

type OCR interface {
	GetOCR(ctx context.Context, url string) (*OCRResult, structured_error.StructuredError)
	Close() error
}

type Describer interface {
	Describe(ctx context.Context, url string) ([]VisionResult, structured_error.StructuredError)
}

type Translator interface {
	Translate(ctx context.Context, message string) (language.Tag, string, structured_error.StructuredError)
	Close() error
}

func logDebugJSON(v interface{}) {
	logrus.DebugFn(func() []interface{} {
		bytes, err := json.Marshal(v)
		if err == nil {
			return []interface{}{string(bytes)}
		}
		return []interface{}{err.Error()}
	})
}
