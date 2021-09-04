package message

import (
	"context"
	"fmt"
	"strings"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

type Localized string

func (l Localized) IsEmpty() bool {
	return string(l) == ""
}

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

func LoadMessages() error {
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
	ocrCommandFormat                 = "get text"
	describeCommandFormat            = "describe"
	noPhotosFormat                   = "I didn't find any photos to interpret, but I appreciate the shoutout!. Try \"@captions_please help\" to learn more"
	wrongMediaFormat                 = "I only know how to interpret photos right now, sorry!"
	imageLabelFormat                 = "Image %d: %s"
	hasAltTextFormat                 = "%s says it's %s"
	noAltTextFormat                  = "%s didn't provide any alt text when posting the image"
	noDescriptionsFormat             = "I'm at a loss for words, sorry!"
	multipleDescriptionsJoinerFormat = "It might also be %s"
	addBotErrorFormat                = "However; %s"
	addDescriptionFormat             = "I think it's %s"
	addOCRFormat                     = "It contains the text: %s"
	unsupportedLanguageFormat        = "I'm unable to support that language right now, sorry!"
	unknownCommandFormat             = "I didn't understand your message, but I appreciate the shoutout! Try \"@captions_please help\" to learn more"
	userBlockedBotCommandFormat      = "I'm blocked from viewing the parent tweet, sorry!"
)

var errorMapping map[structured_error.ErrorType]string = map[structured_error.ErrorType]string{
	structured_error.CannotSplitMessage:  cannotRespondErrorFormat,
	structured_error.NoPhotosFound:       noPhotosFormat,
	structured_error.WrongMediaType:      wrongMediaFormat,
	structured_error.DescribeError:       noDescriptionsFormat,
	structured_error.OCRError:            noDescriptionsFormat,
	structured_error.TranslateError:      noDescriptionsFormat,
	structured_error.UnsupportedLanguage: unsupportedLanguageFormat,
	structured_error.UserBlockedBot:      userBlockedBotCommandFormat,
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

func UnknownCommandMessage(ctx context.Context) Localized {
	return sprint(ctx, unknownCommandFormat)
}

func LabelImage(ctx context.Context, description Localized, index int) Localized {
	return sprintf(ctx, imageLabelFormat, index+1, description)
}

func NoAltText(ctx context.Context, userDisplayName string) Localized {
	return sprintf(ctx, noAltTextFormat, userDisplayName)
}

func HasAltText(ctx context.Context, userDisplayName string, altText string) Localized {
	return sprintf(ctx, hasAltTextFormat, userDisplayName, altText)
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

func AddDescription(ctx context.Context, altText Localized, description Localized) Localized {
	messages := []Localized{altText, sprintf(ctx, addDescriptionFormat, description)}
	return CombineMessages(messages, ". ")
}

func AddBotError(ctx context.Context, altText Localized, err structured_error.StructuredError) Localized {
	errMessage := ErrorMessage(ctx, err)
	return sprintf(ctx, addBotErrorFormat, errMessage)
}

func AddOCR(ctx context.Context, description Localized, ocr Localized) Localized {
	messages := []Localized{description, sprintf(ctx, addOCRFormat, ocr)}
	return CombineMessages(messages, ". ")
}

func GetCompatibleLanguage(ctx context.Context, matcher language.Matcher) (language.Tag, structured_error.StructuredError) {
	tag := language.English
	var err structured_error.StructuredError = nil
	desired := GetLanguage(ctx)
	possibleMatch, _, confidence := matcher.Match(desired)
	if confidence >= language.High {
		tag = possibleMatch
	} else {
		err = structured_error.Wrap(fmt.Errorf("%v has no high confidence matching language", desired), structured_error.UnsupportedLanguage)
	}
	return tag, err
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
	{"en", hasAltTextFormat, catalog.String("%[1]s says it's %[2]s")},
	{"en", addBotErrorFormat, catalog.String("However; %[1]s")},
	{"en", noDescriptionsFormat, noDescriptionsFormat},
	{"en", multipleDescriptionsJoinerFormat, catalog.String("It might also be %[1]s")},
	{"en", addDescriptionFormat, catalog.String("I think it's %[1]s")},
	{"en", addOCRFormat, catalog.String("It contains the text: %[1]s")},
	{"en", unsupportedLanguageFormat, unsupportedLanguageFormat},
	{"en", unknownCommandFormat, unknownCommandFormat},
	{"en", userBlockedBotCommandFormat, userBlockedBotCommandFormat},
	{"de", helpCommandFormat, "Hilfe"},
	{"de", altTextCommandFormat, "Alternativtext"},
	{"de", ocrCommandFormat, "Text scannen"},
	{"de", describeCommandFormat, "beschreiben"},
	{"de", helpUsageFormat, "Markiere @captions_please in einem Tweet, um eine Bildbeschreibung zu bekommen. Füge eines der Kommandos hinzu, wie"},
	{"de", altTextUsageFormat, "Lese, was schon als Bildbeschreibung hinzugefügt ist"},
	{"de", ocrUsageFormat, "Scanne, was an Text im Bild vorhanden ist (Text in Bildform)"},
	{"de", describeUsageFormat, "Nutze KI (Künstliche Intelligenz), um eine Bildbeschreibung zu erzeugen"},
}

func sprint(ctx context.Context, format string) Localized {
	tag := getServerSupportedLanguage(ctx)
	// N.B. If you call printer.Sprint, it doesn't do any translations!!
	// Therefore, call Sprintf without any formatters or specifiers
	return Localized(message.NewPrinter(tag).Sprintf(format))
}

func sprintf(ctx context.Context, format string, args ...interface{}) Localized {
	tag := getServerSupportedLanguage(ctx)
	return Localized(message.NewPrinter(tag).Sprintf(format, args...))
}

func getServerSupportedLanguage(ctx context.Context) language.Tag {
	matcher := language.NewMatcher([]language.Tag{language.English, language.German})
	tag, err := GetCompatibleLanguage(ctx, matcher)
	if err != nil {
		tag = language.English
	}
	logrus.Debug(fmt.Sprintf("Server supported language is %v", tag))
	return tag
}
