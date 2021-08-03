package api

import (
	"context"
	"strings"

	"github.com/AnilRedshift/captions_please_go/internal/api/common"
	"github.com/AnilRedshift/captions_please_go/internal/api/handle_command"
	"github.com/urfave/cli/v2"
)

func handleCommand(ctx context.Context, command string, job activityJob) <-chan common.ActivityResult {
	builder := &strings.Builder{}
	var out <-chan common.ActivityResult
	helpTemplate := `Commands:
{{range .VisibleCommands}}{{join .Names ", "}}{{":\t"}}{{.Usage}}{{"\n"}}{{end}}`
	app := &cli.App{
		Name: "Captions, Please!",
		Commands: []*cli.Command{
			{
				Name:  "help",
				Usage: "Get info about the actions I can take",
				Action: func(c *cli.Context) error {
					err := cli.ShowAppHelp(c)
					if err == nil {
						reply := builder.String()
						out = handle_command.Help(ctx, job.tweet, reply)
					}
					return err
				},
			},
			{
				Name:   "auto",
				Hidden: true,
				Action: func(c *cli.Context) error {
					out = HandleAuto(ctx, job.tweet)
					return nil
				},
			},
			{
				Name:  "alt_text",
				Usage: "See what description the user gave when creating the tweet",
				Action: func(c *cli.Context) error {
					out = HandleAltText(ctx, job.tweet)
					return nil
				},
			},
			{
				Name:  "ocr",
				Usage: "Scan the immage for text",
				Action: func(c *cli.Context) error {
					out = HandleOCR(ctx, job.tweet)
					return nil
				},
			},
			{
				Name:  "describe",
				Usage: "Use AI to create a description of the image",
				Action: func(c *cli.Context) error {
					out = HandleDescribe(ctx, job.tweet)
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
		result := common.ActivityResult{Tweet: job.tweet, Err: err, Action: "handle command"}
		out <- result
		close(out)
	}

	if out == nil {
		panic("handleCommand returning nil out")
	}
	return out
}
