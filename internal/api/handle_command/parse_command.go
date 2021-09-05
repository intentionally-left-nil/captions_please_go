package handle_command

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
)

type command struct {
	auto      bool
	help      bool
	altText   bool
	ocr       bool
	describe  bool
	unknown   bool
	translate bool
	tag       language.Tag
}

func (c *command) isEmpty() bool {
	return !(c.auto || c.help || c.altText || c.ocr || c.describe || c.unknown)
}

func parseCommand(message string) command {
	message = strings.TrimSpace(strings.ToLower(message))
	message = strings.ReplaceAll(message, ",", "")
	tokens := strings.Fields(message)
	logrus.Debug(fmt.Sprintf("parseCommand parsing tokens %v", tokens))
	c := parseGerman(tokens)
	if c != nil {
		return *c
	}

	c = parseEnglish(tokens)
	if c == nil {
		c = &command{unknown: true, tag: language.English}
	}
	return *c
}

func parseGerman(tokens []string) *command {
	c := &command{tag: language.German}
	tokens = parseGermanRemoveModifiers(tokens)
	parseGermanDirectives(c, tokens)
	if c.isEmpty() {
		c = nil
	}
	return c
}

func parseGermanRemoveModifiers(tokens []string) []string {
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		switch token {
		case "und":
		case "das":
		default:
			filtered = append(filtered, token)
		}
	}
	return filtered
}

func parseGermanDirectives(c *command, tokens []string) (remainder []string) {
	if len(tokens) >= 1 {
		foundToken := true
		switch tokens[0] {
		case "hilfe":
			c.help = true
		case "alternativtext":
			c.altText = true
		case "scannen":
			c.ocr = true
		case "beschreiben":
			c.describe = true
		case "text":
		default:
			foundToken = false
		}

		if foundToken {
			remainder = parseGermanDirectives(c, tokens[1:])
		}
	} else {
		remainder = tokens
	}
	return remainder
}

func parseEnglish(tokens []string) *command {
	c := &command{}
	if len(tokens) == 0 {
		// Special case for English - no text == auto in english
		c = &command{auto: true, tag: language.English}
	} else {
		remainder := parseEnglishRemoveModifiers(tokens)
		tag, remainder := parseEnglishLang(remainder)
		remainder = parseEnglishDirectives(c, remainder)

		if tag == nil {
			tag, _ = parseEnglishLang(remainder)
		}
		if tag == nil {
			tag = &language.English
		}
		c.tag = *tag

		// Special case for English,tag but no directive = auto in that language
		if c.isEmpty() && tag != nil && len(remainder) == 0 {
			c = &command{auto: true, tag: *tag}
		}
	}

	if c.isEmpty() {
		c = nil
	}
	return c
}

func parseEnglishRemoveModifiers(tokens []string) []string {
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		switch token {
		case "and":
		case "the":
		default:
			filtered = append(filtered, token)
		}
	}
	return filtered
}

func parseEnglishDirectives(c *command, tokens []string) (remainder []string) {
	remainder = tokens
	if len(tokens) >= 1 {
		foundToken := true
		switch tokens[0] {
		case "help":
			c.help = true
			remainder = remainder[1:]
		case "auto":
			c.auto = true
			remainder = remainder[1:]
		case "text":
			fallthrough
		case "ocr":
			c.ocr = true
			remainder = remainder[1:]
		case "describe":
			fallthrough
		case "caption":
			c.describe = true
			remainder = remainder[1:]
		case "alttext":
			fallthrough
		case "alt_text":
			c.altText = true
			remainder = remainder[1:]
		case "alt":
			if len(tokens) >= 2 && tokens[1] == "text" {
				c.altText = true
				remainder = remainder[2:]
			}
		case "get":
			remainder = remainder[1:]
		default:
			foundToken = false
		}
		if foundToken {
			remainder = parseEnglishDirectives(c, remainder)
		}
	}

	return remainder
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
