package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/AnilRedshift/captions_please_go/internal/api"
	"github.com/AnilRedshift/captions_please_go/pkg/vision"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {

	app := &cli.App{
		Name: "vision",
		Commands: []*cli.Command{
			{
				Name:   "ocr",
				Usage:  "Get the document text of an image",
				Action: ocr,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "provider", Value: "google"},
					&cli.StringFlag{Name: "url", Required: true},
				},
			},
			{
				Name:   "caption",
				Usage:  "Get a ML generated caption of an image",
				Action: caption,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "provider", Value: "azure"},
					&cli.StringFlag{Name: "url", Required: true},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "verbose"},
		},
		Before: onBefore,
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}

func onBefore(c *cli.Context) error {
	if c.Bool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	return nil
}

func ocr(c *cli.Context) error {
	secrets, err := api.NewSecrets()
	if err == nil {
		var ocr vision.OCR
		switch c.String("provider") {
		case "google":
			ocr, err = vision.NewGoogleVision(secrets.GooglePrivateKeyID, secrets.GooglePrivateKeySecret)
		case "azure":
			ocr = vision.NewAzureVision(secrets.AzureComputerVisionKey).(vision.OCR)
		default:
			err = errors.New("invalid provider, must be [google|azure]")
		}

		if err == nil {
			var result *vision.OCRResult
			result, err = ocr.GetOCR(c.String("url"))
			if err == nil {
				printJSON(result)
			}
		}
	}
	return err
}

func caption(c *cli.Context) error {
	secrets, err := api.NewSecrets()
	if err == nil {
		var describer vision.Describer
		switch c.String("provider") {
		case "azure":
			describer = vision.NewAzureVision(secrets.AzureComputerVisionKey)
		case "google":
			err = errors.New("google is not a supported provider for image captions")
		default:
			err = errors.New("invalid provider, must be [google|azure]")
		}
		if err == nil {
			results, err := describer.Describe(c.String("url"))
			if err == nil {
				for _, result := range results {
					printJSON(result)
				}
			}

		}
	}
	return err
}

func printJSON(v interface{}) {
	message, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(message))
}
