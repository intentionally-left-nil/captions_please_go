package handle_command

import (
	"context"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
)

func Command(ctx context.Context, message string, job common.ActivityJob) common.ActivityResult {
	command := parseCommand(message)
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
