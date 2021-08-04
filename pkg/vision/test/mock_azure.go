package vision_test

import (
	"testing"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/vision"
	"github.com/stretchr/testify/assert"
)

type MockAzure struct {
	T            *testing.T
	DescribeMock func(url string) ([]vision.VisionResult, error)
}

func (a *MockAzure) Describe(url string) ([]vision.VisionResult, structured_error.StructuredError) {
	assert.NotNil(a.T, a.DescribeMock)
	result, err := a.DescribeMock(url)
	return result, structured_error.Wrap(err, structured_error.DescribeError)
}
