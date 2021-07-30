package vision

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type OCRLanguage struct {
	Code       string
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
	GetOCR(url string) (*OCRResult, error)
	Close() error
}

type Describer interface {
	Describe(url string) ([]VisionResult, error)
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
