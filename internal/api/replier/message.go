package replier

import (
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

type Message string

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
	unknownErrorTag       = "My joints are freezing up! Hey @TheOtherAnil can you please fix me?"
	cannotRespondErrorTag = "The message can't be written out as a tweet. Maybe it's by Prince?"
)

func UnknownError(tag language.Tag) Message {
	return sprint(tag, unknownErrorTag)
}

func CannotRespondError(tag language.Tag) Message {
	return sprint(tag, cannotRespondErrorTag)
}

func GetErrorMessage(err structured_error.StructuredError, tag language.Tag) Message {
	switch err.Type() {
	default:
		return UnknownError(tag)
	}
}

var messages = [...]struct {
	tag     string
	format  string
	message interface{}
}{
	{"en", unknownErrorTag, unknownErrorTag},
	{"en", cannotRespondErrorTag, cannotRespondErrorTag},
}

func sprint(tag language.Tag, format string) Message {
	return Message(message.NewPrinter(tag).Sprint(format))
}
