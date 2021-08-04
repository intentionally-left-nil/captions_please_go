package handle_command

import (
	"context"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/internal/api/replier"
)

func Command(ctx context.Context, message string, job common.ActivityJob) common.ActivityResult {
	command := parseCommand(message)
	ctx = replier.WithLanguage(ctx, command.tag)

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
		fallthrough
	default:
		return Help(ctx, job.Tweet)
	}
}
