package replier

import (
	"errors"
	"testing"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"
)

func TestLoadMessages(t *testing.T) {
	assert.NoError(t, loadMessages())
}

func TestGetErrorMessage(t *testing.T) {
	assert.NoError(t, loadMessages())
	anError := errors.New("oh no")
	tests := []struct {
		name     string
		err      structured_error.StructuredError
		enResult Localized
	}{
		{
			name:     "Defaults to an unknown error",
			err:      structured_error.Wrap(anError, structured_error.ErrorType(999)),
			enResult: unknownErrorFormat,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.enResult, ErrorMessage(test.err, language.AmericanEnglish))
		})
	}
}

func TestLabelImage(t *testing.T) {
	assert.NoError(t, loadMessages())
	assert.Equal(t, Localized("Image 1: foo"), LabelImage(language.AmericanEnglish, Unlocalized("foo"), 0))
	assert.Equal(t, Localized("Image 2: foo"), LabelImage(language.AmericanEnglish, Unlocalized("foo"), 1))
}
