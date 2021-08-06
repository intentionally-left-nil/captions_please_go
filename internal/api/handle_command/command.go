package handle_command

import (
	"context"
	"fmt"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/pkg/message"
	"github.com/sirupsen/logrus"
)

func Command(ctx context.Context, toParse string, job common.ActivityJob) common.ActivityResult {
	command := parseCommand(toParse)
	logrus.Debug(fmt.Sprintf("%s: Directive %s for language %v", job.Tweet.Id, command.directive, command.tag))
	ctx = message.WithLanguage(ctx, command.tag)

	switch command.directive {
	case autoDirective:
		return HandleAuto(ctx, job.Tweet)
	case altTextDirective:
		return HandleAltText(ctx, job.Tweet)
	case ocrDirective:
		return HandleOCR(ctx, job.Tweet)
	case describeDirective:
		return HandleDescribe(ctx, job.Tweet)
	case helpDirective:
		return Help(ctx, job.Tweet)
	case unknownDirective:
		fallthrough
	default:
		return Unknown(ctx, job.Tweet)
	}
}
