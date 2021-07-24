package vision

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
}

type Describer interface {
	Describe(url string) ([]VisionResult, error)
}
