# Captions, Please!

Welcome to the source code powering the @captions_please twitter bot.

## Translations needed

Do you speak English and another language? I am looking to add localized translations to @captions_please, and need your help! Here are the strings that need translating:

| English                                                                                                          | Description                                                                                                                              | Placeholders                                                     |
| ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| help                                                                                                             | As in @captions_please help                                                                                                              | None                                                             |
| You can customize the response by adding one of the following commands after tagging me:`                        | The preamble of the help response when a user types @captions_please help                                                                | None                                                             |
| alt text                                                                                                         | As in @captions_please alt text. Gets the user-generated alt text                                                                        | None                                                             |
| See what description the user gave when creating the tweet                                                       | A help message to describe what happens when you respond with @captions_please alt text                                                  | None                                                             |
| ocr                                                                                                              | As in @captions_please ocr. Returns any text in the image                                                                                | None                                                             |
| Scan the image for text                                                                                          | A help message to describe what happens when you call @captions_please ocr                                                               | None                                                             |
| describe                                                                                                         | As in @captions_please describe. A command telling the bot to generate a caption visually describing the image                           | None                                                             |
| Use AI to create a description of the image                                                                      | A help message to describe what happens when you call @captions_please describe                                                          | None                                                             |
| My joints are freezing up! Hey @TheOtherAnil can you please fix me?                                              | A witty message (doesn't need to translate exactly) indicating an error occured                                                          | None                                                             |
| The message can't be written out as a tweet. Maybe it's by Prince?                                               | A witty message (doesn't need to translate exactly) indicating there was some issue replying to the tweet                                | None                                                             |
| I didn't find any photos to interpret, but I appreciate the shoutout!. Try "@captions_please help" to learn more | An error message if the bot couldn't find any tweets with images to scan                                                                 | None                                                             |
| I only know how to interpret photos right now, sorry!                                                            | An error message if you try to tag @captions_please on a video, or a gif                                                                 | None                                                             |
| Image %d: %s                                                                                                     | For multiple images, the bot wants to reply with Image 1: a caption. Image 2: some other caption. This joins "Image N:" with the caption | %d: The image number. %s: The caption for the image              |
| %s didn't provide any alt text when posting the image                                                            | An error message if the user didn't include any alt text                                                                                 | %s is the Display name of the user who posted the original image |
| I'm at a loss for words, sorry!                                                                                  | Error when the bot couldn't come up with a description for an image                                                                      | None                                                             |
| It might also be %s                                                                                              | A way to combine multiple descriptions. For example: It's a bird. It might also be a plane                                               | %s is the caption that could also apply                          |
| It contains the text: %s                                                                                         | A prefix for OCR results. For example: It contains the text original pretz baked snack sticks                                            | %s is the OCR text contents to join                              |

## How it works

The bot registers to Twitter with the [account activity api](https://developer.twitter.com/en/docs/twitter-api/enterprise/account-activity-api/overview) to receive notifications when users interact with the @captions_please bot
It then uses more twitter API's to find the image(s) the user wants to know more about.

Then, if needed, it queries [azure cognitive services](https://docs.microsoft.com/en-us/azure/cognitive-services/computer-vision/tutorials/storage-lab-tutorial) or [google cloud vision](https://cloud.google.com/vision/docs/samples/vision-document-text-tutorial) to generate the captions.
The captions are then returned to the user as a series of tweets

## Running the bot

```bash
cd ./cmd/captions_please
go build .
./captions_please --verbose
```

## Local development

First, a caveat: This is my first real program written in Golang. Some of the patterns chosen were explicit attempts to learn about fundamentals, such as channels.
That said, the program is organized into a few main pieces: /pkg contains the API wrappers around Twitter, Azure, and Google Cloud API. /internal/api handles the webhooks
/internal/api/handle_command handles the specific logic of each bot command and /internal/api/replier handles internationalization & replying to twitter
