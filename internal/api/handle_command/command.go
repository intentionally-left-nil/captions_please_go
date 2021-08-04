package handle_command

import (
	"context"
	"strings"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/urfave/cli/v2"
)

func Command(ctx context.Context, command string, job common.ActivityJob) common.ActivityResult {
	builder := &strings.Builder{}
	var result common.ActivityResult
	helpTemplate := `Commands:
{{range .VisibleCommands}}{{join .Names ", "}}{{":\t"}}{{.Usage}}{{"\n"}}{{end}}`
	app := &cli.App{
		Name: "Captions, Please!",
		Commands: []*cli.Command{
			{
				Name:  "help",
				Usage: "Get info about the actions I can take",
				Action: func(c *cli.Context) error {
					result = Help(ctx, job.Tweet)
					return nil
				},
			},
			{
				Name:   "auto",
				Hidden: true,
				Action: func(c *cli.Context) error {
					result = HandleAuto(ctx, job.Tweet)
					return nil
				},
			},
			{
				Name:  "alt_text",
				Usage: "See what description the user gave when creating the tweet",
				Action: func(c *cli.Context) error {
					result = HandleAltText(ctx, job.Tweet)
					return nil
				},
			},
			{
				Name:  "ocr",
				Usage: "Scan the immage for text",
				Action: func(c *cli.Context) error {
					result = HandleOCR(ctx, job.Tweet)
					return nil
				},
			},
			{
				Name:  "describe",
				Usage: "Use AI to create a description of the image",
				Action: func(c *cli.Context) error {
					result = HandleDescribe(ctx, job.Tweet)
					return nil
				},
			},
		},
		CustomAppHelpTemplate: helpTemplate,
		Writer:                builder,
	}
	err := app.Run(strings.Split("captions_please "+command, " "))
	if err != nil {
		out := make(chan common.ActivityResult, 1)
		result := common.ActivityResult{Tweet: job.Tweet, Err: err, Action: "handle command"}
		out <- result
		close(out)
	}

	return result
}
