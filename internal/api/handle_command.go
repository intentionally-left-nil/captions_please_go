package api

import (
	"strings"

	"github.com/AnilRedshift/captions_please_go/pkg/twitter"
	"github.com/urfave/cli/v2"
)

func handleCommand(command string, tweet *twitter.Tweet) error {
	builder := &strings.Builder{}
	helpTemplate := `Commands:
{{range .VisibleCommands}}{{join .Names ", "}}{{":\t"}}{{.Usage}}{{"\n"}}{{end}}`
	app := &cli.App{
		Name: "Captions, Please!",
		Commands: []*cli.Command{
			{
				Name:  "help",
				Usage: "Get info about the actions I can take",
				Action: func(c *cli.Context) error {
					return cli.ShowAppHelp(c)
				},
			},
			{
				Name:   "auto",
				Hidden: true,
			},
		},
		CustomAppHelpTemplate: helpTemplate,
		Writer:                builder,
	}
	err := app.Run(strings.Split("captions_please "+command, " "))
	return err
}
