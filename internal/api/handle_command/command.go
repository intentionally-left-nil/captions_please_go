package handle_command

import (
	"context"
	"errors"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
)

func Command(ctx context.Context, toParse string, job common.ActivityJob) common.ActivityResult {

	didPanic := true
	defer func() {
		if didPanic {
			replyWithError(ctx, job.Tweet, structured_error.Wrap(errors.New("panic at the disco"), structured_error.Unknown))
		}
	}()

	command := parseCommand(toParse)
	ctx = message.WithLanguage(ctx, command.tag)

	var result common.ActivityResult
	if command.auto {
		result = HandleAuto(ctx, job.Tweet)
	} else if command.altText {
		result = HandleAltText(ctx, job.Tweet)
	} else if command.ocr {
		result = HandleOCR(ctx, job.Tweet)
	} else if command.describe {
		result = HandleDescribe(ctx, job.Tweet)
	} else if command.help {
		result = Help(ctx, job.Tweet)
	} else if command.unknown {
		result = Unknown(ctx, job.Tweet)
	} else {
		result = Unknown(ctx, job.Tweet)
	}
	didPanic = false
	return result
}
