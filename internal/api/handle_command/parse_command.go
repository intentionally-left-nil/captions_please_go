package handle_command

import (
	"strings"

	"golang.org/x/text/language"
)

type directive int

const (
	autoDirective directive = iota
	helpDirective
	altTextDirective
	ocrDirective
	describeDirective
)

type command struct {
	directive directive
	tag       language.Tag
}

func parseCommand(message string) command {
	message = strings.ToLower(message)
	message = strings.ReplaceAll(message, ",", "")
	tokens := strings.Fields(message)
	c := parseEnglish(tokens)
	if c == nil {
		c = &command{directive: helpDirective, tag: language.English}
	}
	return *c
}

func parseEnglish(tokens []string) *command {
	var c *command
	if len(tokens) == 0 {
		// Special case for English - no text == auto in english
		c = &command{directive: autoDirective, tag: language.English}
	} else {
		remainder := tokens
		tag, remainder := parseEnglishLang(remainder)
		dir, remainder := parseEnglishDirective(remainder)
		if tag == nil {
			tag, _ = parseEnglishLang(remainder)
		}

		// Special case for English,tag but no directive = auto in that language
		if dir == nil && tag != nil {
			c = &command{directive: autoDirective, tag: *tag}
		} else if dir != nil {
			if tag == nil {
				tag = &language.English
			}
			c = &command{tag: *tag, directive: *dir}
		}
	}
	return c
}

func parseEnglishDirective(tokens []string) (*directive, []string) {
	var d *directive
	remainder := tokens
	if len(tokens) >= 1 {
		switch tokens[0] {
		case "help":
			dir := helpDirective
			d = &dir
			remainder = remainder[1:]
		case "auto":
			dir := autoDirective
			d = &dir
			remainder = remainder[1:]
		case "ocr":
			dir := ocrDirective
			d = &dir
			remainder = remainder[1:]
		case "describe":
			fallthrough
		case "caption":
			dir := describeDirective
			d = &dir
			remainder = remainder[1:]
		case "alttext":
			fallthrough
		case "alt_text":
			dir := altTextDirective
			d = &dir
			remainder = remainder[1:]
		case "alt":
			if len(tokens) >= 2 && tokens[1] == "text" {
				dir := altTextDirective
				d = &dir
				remainder = remainder[2:]
			}
		}
	}

	return d, remainder
}

func parseEnglishLang(tokens []string) (*language.Tag, []string) {
	var tag *language.Tag
	remainder := tokens
	if len(tokens) >= 2 && tokens[0] == "in" {
		tag, remainder = parseTag(tokens[1:])
	}
	return tag, remainder
}

var languageMapping = map[string]language.Tag{
	"english": language.English,
	"german":  language.German,
}

func parseTag(tokens []string) (*language.Tag, []string) {
	var tag *language.Tag
	remainder := tokens
	if len(tokens) > 0 {
		token := tokens[0]
		possibleTag, err := language.Parse(token)
		if err == nil {
			tag = &possibleTag
			remainder = remainder[1:]
		} else if possibleTag, ok := languageMapping[token]; ok {
			tag = &possibleTag
			remainder = remainder[1:]
		}
	}
	return tag, remainder
}
