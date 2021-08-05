package replier

import (
	"context"
	"strings"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

type Localized string

type messageCtxKey int

const theMessageKey messageCtxKey = 0

func WithLanguage(ctx context.Context, tag language.Tag) context.Context {
	return context.WithValue(ctx, theMessageKey, tag)
}

func GetLanguage(ctx context.Context) language.Tag {
	if tag, ok := ctx.Value(theMessageKey).(language.Tag); ok {
		return tag
	}
	return language.English
}

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
	helpCommandFormat                = "help"
	altTextCommandFormat             = "alt text"
	ocrCommandFormat                 = "ocr"
	describeCommandFormat            = "describe"
	noPhotosFormat                   = "I didn't find any photos to interpret, but I appreciate the shoutout!. Try \"@captions_please help\" to learn more"
	wrongMediaFormat                 = "I only know how to interpret photos right now, sorry!"
	imageLabelFormat                 = "Image %d: %s"
	noAltTextFormat                  = "%s didn't provide any alt text when posting the image"
	noDescriptionsFormat             = "I'm at a loss for words, sorry!"
	multipleDescriptionsJoinerFormat = "It might also be %s"
	combineDescriptionAndOCRFormat   = "It contains the text: %s"
)

var errorMapping map[structured_error.ErrorType]string = map[structured_error.ErrorType]string{
	structured_error.CannotSplitMessage: cannotRespondErrorFormat,
	structured_error.NoPhotosFound:      noPhotosFormat,
	structured_error.WrongMediaType:     wrongMediaFormat,
	structured_error.DescribeError:      noDescriptionsFormat,
	structured_error.OCRError:           noDescriptionsFormat,
}

func ErrorMessage(ctx context.Context, err structured_error.StructuredError) Localized {
	format, ok := errorMapping[err.Type()]
	if !ok {
		format = unknownErrorFormat
	}
	return sprint(ctx, format)
}

func HelpMessage(ctx context.Context) Localized {
	lines := [][]string{
		{altTextCommandFormat, altTextUsageFormat},
		{ocrCommandFormat, ocrUsageFormat},
		{describeCommandFormat, describeUsageFormat},
	}
	builder := &strings.Builder{}
	builder.WriteString(string(sprint(ctx, helpUsageFormat)))
	for _, formats := range lines {
		builder.WriteString("\n")
		builder.WriteString(string(sprint(ctx, formats[0])))
		builder.WriteString(": ")
		builder.WriteString(string(sprint(ctx, formats[1])))
	}
	return Localized(builder.String())
}

func LabelImage(ctx context.Context, description Localized, index int) Localized {
	return sprintf(ctx, imageLabelFormat, index+1, description)
}
func NoAltText(ctx context.Context, userDisplayName string) Localized {
	return sprintf(ctx, noAltTextFormat, userDisplayName)
}

func CombineMessages(messages []Localized, joiner string) Localized {
	asStrings := make([]string, len(messages))
	for i, message := range messages {
		asStrings[i] = string(message)
	}
	return Localized(strings.Join(asStrings, joiner))
}

func CombineDescriptions(ctx context.Context, descriptions []string) Localized {
	messages := make([]Localized, len(descriptions))
	for i, description := range descriptions {
		if i == 0 {
			messages[i] = Unlocalized(description)

		} else {
			messages[i] = sprintf(ctx, multipleDescriptionsJoinerFormat, description)
		}
	}
	return CombineMessages(messages, ". ")
}

func CombineDescriptionAndOCR(ctx context.Context, description Localized, ocr Localized) Localized {
	messages := []Localized{description, sprintf(ctx, combineDescriptionAndOCRFormat, ocr)}
	return CombineMessages(messages, ". ")
}

func Unlocalized(message string) Localized {
	return Localized(message)
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
	{"en", helpCommandFormat, helpCommandFormat},
	{"en", altTextCommandFormat, altTextCommandFormat},
	{"en", ocrCommandFormat, ocrCommandFormat},
	{"en", describeCommandFormat, describeCommandFormat},
	{"en", noPhotosFormat, noPhotosFormat},
	{"en", wrongMediaFormat, wrongMediaFormat},
	{"en", imageLabelFormat, catalog.String("Image %[1]d: %[2]s")},
	{"en", noAltTextFormat, catalog.String("%[1]s didn't provide any alt text when posting the image")},
	{"en", noDescriptionsFormat, noDescriptionsFormat},
	{"en", multipleDescriptionsJoinerFormat, catalog.String("It might also be %[1]s")},
	{"en", combineDescriptionAndOCRFormat, catalog.String("It contains the text: %[1]s")},
}

func sprint(ctx context.Context, format string) Localized {
	tag := GetLanguage(ctx)
	return Localized(message.NewPrinter(tag).Sprint(format))
}

func sprintf(ctx context.Context, format string, args ...interface{}) Localized {
	tag := GetLanguage(ctx)
	return Localized(message.NewPrinter(tag).Sprintf(format, args...))
}
