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

func TestSimpleErrors(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(language.Tag) localized
		enResult localized
	}{
		{
			name:     "UnknownError",
			fn:       UnknownError,
			enResult: unknownErrorFormat,
		},
		{
			name:     "CannotRespondError",
			fn:       CannotRespondError,
			enResult: cannotRespondErrorFormat,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tag := language.AmericanEnglish
			assert.Equal(t, test.enResult, test.fn(tag))
		})
	}
}

func TestGetErrorMessage(t *testing.T) {
	anError := errors.New("oh no")
	tests := []struct {
		name     string
		err      structured_error.StructuredError
		enResult localized
	}{
		{
			name:     "Defaults to an unknown error",
			err:      structured_error.Wrap(anError, structured_error.ErrorType(999)),
			enResult: unknownErrorFormat,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.enResult, GetErrorMessage(test.err, language.AmericanEnglish))
		})
	}
}
