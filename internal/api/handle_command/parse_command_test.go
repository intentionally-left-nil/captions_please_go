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
			expected: command{directive: autoDirective, tag: language.English},
		},
		{
			command:  " ",
			expected: command{directive: autoDirective, tag: language.English},
		},
		{
			command:  "this is some random text",
			expected: command{directive: helpDirective, tag: language.English},
		},
		{
			command:  "in english",
			expected: command{directive: autoDirective, tag: language.English},
		},
		{
			command:  "in en",
			expected: command{directive: autoDirective, tag: language.English},
		},
		{
			command:  "in en-US",
			expected: command{directive: autoDirective, tag: language.AmericanEnglish},
		},
		{
			command:  "in german",
			expected: command{directive: autoDirective, tag: language.German},
		},
		{
			command:  "in de",
			expected: command{directive: autoDirective, tag: language.German},
		},
		{
			command:  "Help",
			expected: command{directive: helpDirective, tag: language.English},
		},
		{
			command:  "ocr",
			expected: command{directive: ocrDirective, tag: language.English},
		},
		{
			command:  "auto",
			expected: command{directive: autoDirective, tag: language.English},
		},
		{
			command:  "describe",
			expected: command{directive: describeDirective, tag: language.English},
		},
		{
			command:  "caption",
			expected: command{directive: describeDirective, tag: language.English},
		},
		{
			command:  "alt_text",
			expected: command{directive: altTextDirective, tag: language.English},
		},
		{
			command:  "AltText",
			expected: command{directive: altTextDirective, tag: language.English},
		},
		{
			command:  "alt text",
			expected: command{directive: altTextDirective, tag: language.English},
		},
		{
			command:  "alt text in english",
			expected: command{directive: altTextDirective, tag: language.English},
		},
		{
			command:  "alt text in german",
			expected: command{directive: altTextDirective, tag: language.German},
		},
		{
			command:  "alttext in german",
			expected: command{directive: altTextDirective, tag: language.German},
		},
		{
			command:  "in german, alt text",
			expected: command{directive: altTextDirective, tag: language.German},
		},
	}

	for _, test := range tests {
		t.Run(test.command, func(t *testing.T) {
			assert.Equal(t, test.expected, parseCommand(test.command))
		})
	}
}
