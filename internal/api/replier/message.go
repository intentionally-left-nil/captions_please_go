package replier

import (
	"strings"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

type localized string

func loadMessages() error {
	var err error
	for _, entry := range messages {
		var tag language.Tag
		tag, err = language.Parse(entry.tag)
		if err == nil {
			switch msg := entry.message.(type) {
			case string:
				err = message.SetString(tag, entry.format, msg)
			case catalog.Message:
				err = message.Set(tag, entry.format, msg)
			case []catalog.Message:
				err = message.Set(tag, entry.format, msg...)
			}
		}
		if err != nil {
			break
		}
	}
	return err
}

const (
	unknownErrorFormat       = "My joints are freezing up! Hey @TheOtherAnil can you please fix me?"
	cannotRespondErrorFormat = "The message can't be written out as a tweet. Maybe it's by Prince?"
	altTextUsageFormat       = "See what description the user gave when creating the tweet"
	ocrUsageFormat           = "Scan the image for text"
	describeUsageFormat      = "Use AI to create a description of the image"
	helpUsageFormat          = `Tag @captions_please in a tweet to interpret the images.
You can customize the response by adding one of the following commands after tagging me:`
	helpCommandFormat     = "help"
	altTextCommandFormat  = "alt_text"
	ocrCommandFormat      = "ocr"
	describeCommandFormat = "describe"
)

func UnknownError(tag language.Tag) localized {
	return sprint(tag, unknownErrorFormat)
}

func CannotRespondError(tag language.Tag) localized {
	return sprint(tag, cannotRespondErrorFormat)
}

func GetErrorMessage(err structured_error.StructuredError, tag language.Tag) localized {
	switch err.Type() {
	default:
		return UnknownError(tag)
	}
}

func HelpMessage(tag language.Tag) localized {
	lines := [][]string{
		{altTextCommandFormat, altTextUsageFormat},
		{ocrCommandFormat, ocrUsageFormat},
		{describeCommandFormat, describeUsageFormat},
	}
	builder := &strings.Builder{}
	builder.WriteString(string(sprint(tag, helpUsageFormat)))
	for _, formats := range lines {
		builder.WriteString("\n")
		builder.WriteString(string(sprint(tag, formats[0])))
		builder.WriteString(":\t")
		builder.WriteString(string(sprint(tag, formats[1])))
	}
	return localized(builder.String())
}

func Unlocalized(message string) localized {
	return localized(message)
}

var messages = [...]struct {
	tag     string
	format  string
	message interface{}
}{
	{"en", unknownErrorFormat, unknownErrorFormat},
	{"en", cannotRespondErrorFormat, cannotRespondErrorFormat},
	{"en", altTextUsageFormat, altTextUsageFormat},
	{"en", ocrUsageFormat, ocrUsageFormat},
	{"en", describeUsageFormat, describeUsageFormat},
	{"en", helpUsageFormat, helpUsageFormat},
}

func sprint(tag language.Tag, format string) localized {
	return localized(message.NewPrinter(tag).Sprint(format))
}
