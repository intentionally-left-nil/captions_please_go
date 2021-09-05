package handle_command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		command  string
		expected command
	}{
		{
			command:  "",
			expected: command{auto: true, tag: language.English},
		},
		{
			command:  " ",
			expected: command{auto: true, tag: language.English},
		},
		{
			command:  "translate",
			expected: command{auto: true, translate: true, tag: language.English},
		},
		{
			command:  "this is some random text",
			expected: command{unknown: true, tag: language.English},
		},
		{
			command:  "in english",
			expected: command{auto: true, tag: language.English},
		},
		{
			command:  "into english",
			expected: command{auto: true, tag: language.English},
		},
		{
			command:  "translate into english",
			expected: command{auto: true, translate: true, tag: language.English},
		},
		{
			command:  "translate into german",
			expected: command{auto: true, translate: true, tag: language.German},
		},
		{
			command:  "in en",
			expected: command{auto: true, tag: language.English},
		},
		{
			command:  "in en-US",
			expected: command{auto: true, tag: language.AmericanEnglish},
		},
		{
			command:  "in german",
			expected: command{auto: true, tag: language.German},
		},
		{
			command:  "in de",
			expected: command{auto: true, tag: language.German},
		},
		{
			command:  "Help",
			expected: command{help: true, tag: language.English},
		},
		{
			command:  "ocr",
			expected: command{ocr: true, tag: language.English},
		},
		{
			command:  "ocr and translate",
			expected: command{ocr: true, translate: true, tag: language.English},
		},
		{
			command:  "text",
			expected: command{ocr: true, tag: language.English},
		},
		{
			command:  "get text",
			expected: command{ocr: true, tag: language.English},
		},
		{
			command:  "auto",
			expected: command{auto: true, tag: language.English},
		},
		{
			command:  "describe",
			expected: command{describe: true, tag: language.English},
		},
		{
			command:  "caption",
			expected: command{describe: true, tag: language.English},
		},
		{
			command:  "alt_text",
			expected: command{altText: true, tag: language.English},
		},
		{
			command:  "AltText",
			expected: command{altText: true, tag: language.English},
		},
		{
			command:  "alt text",
			expected: command{altText: true, tag: language.English},
		},
		{
			command:  "alt text in english",
			expected: command{altText: true, tag: language.English},
		},
		{
			command:  "get text and describe",
			expected: command{ocr: true, describe: true, tag: language.English},
		},
		{
			command:  "alt text, get text, and describe in english",
			expected: command{ocr: true, altText: true, describe: true, tag: language.English},
		},
		{
			command:  "alt text, get text, describe, and translate into english",
			expected: command{ocr: true, altText: true, describe: true, translate: true, tag: language.English},
		},
		{
			command:  "alt text in german",
			expected: command{altText: true, tag: language.German},
		},
		{
			command:  "alttext in german",
			expected: command{altText: true, tag: language.German},
		},
		{
			command:  "in german, alt text",
			expected: command{altText: true, tag: language.German},
		},
		{
			command:  "in german, get alt text",
			expected: command{altText: true, tag: language.German},
		},
		{
			command:  "hilfe",
			expected: command{help: true, tag: language.German},
		},
		{
			command:  "AlternativText",
			expected: command{altText: true, tag: language.German},
		},
		{
			command:  "Scannen und beschreiben",
			expected: command{ocr: true, describe: true, tag: language.German},
		},
		{
			command:  "Scannen",
			expected: command{ocr: true, tag: language.German},
		},
		{
			command:  "Text scannen",
			expected: command{ocr: true, tag: language.German},
		},
		{
			command:  "beschreiben",
			expected: command{describe: true, tag: language.German},
		},
		{
			command:  "Text beschreiben",
			expected: command{describe: true, tag: language.German},
		},
	}

	for _, test := range tests {
		t.Run(test.command, func(t *testing.T) {
			assert.Equal(t, test.expected, parseCommand(test.command))
		})
	}
}
