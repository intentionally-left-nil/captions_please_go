package api

import (
	"context"
	"strings"

	"github.com/urfave/cli/v2"
)

func handleCommand(ctx context.Context, command string, job activityJob) <-chan ActivityResult {
	builder := &strings.Builder{}
	var out <-chan ActivityResult
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
						out = HandleHelp(ctx, job.tweet, reply)
					}
					return err
				},
			},
			{
				Name:   "auto",
				Hidden: true,
			},
			{
				Name:  "alt_text",
				Usage: "See what description the user gave when creating the tweet",
				Action: func(c *cli.Context) error {
					out = HandleAltText(ctx, job.tweet)
					return nil
				},
			},
		},
		CustomAppHelpTemplate: helpTemplate,
		Writer:                builder,
	}
	err := app.Run(strings.Split("captions_please "+command, " "))
	if err != nil {
		out := make(chan ActivityResult, 1)
		result := ActivityResult{tweet: job.tweet, err: err, action: "handle command"}
		out <- result
		close(out)
	}

	if out == nil {
		panic("handleCommand returning nil out")
	}
	return out
}
