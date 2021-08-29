package handle_command

import (
	"context"
	"errors"
	"fmt"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/sirupsen/logrus"
)

func Command(ctx context.Context, toParse string, job common.ActivityJob) common.ActivityResult {

	didPanic := true
	defer func() {
		if didPanic {
			replyWithError(ctx, job.Tweet, structured_error.Wrap(errors.New("panic at the disco"), structured_error.Unknown))
		}
	}()

	command := parseCommand(toParse)
	logrus.Debug(fmt.Sprintf("%s: Directive %s for language %v", job.Tweet.Id, command.directive, command.tag))
	ctx = message.WithLanguage(ctx, command.tag)

	var result common.ActivityResult
	switch command.directive {
	case autoDirective:
		result = HandleAuto(ctx, job.Tweet)
	case altTextDirective:
		result = HandleAltText(ctx, job.Tweet)
	case ocrDirective:
		result = HandleOCR(ctx, job.Tweet)
	case describeDirective:
		result = HandleDescribe(ctx, job.Tweet)
	case helpDirective:
		result = Help(ctx, job.Tweet)
	case unknownDirective:
		fallthrough
	default:
		result = Unknown(ctx, job.Tweet)
	}
	didPanic = false
	return result
}
