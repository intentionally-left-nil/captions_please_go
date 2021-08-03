package common

import "github.com/AnilRedshift/captions_please_go/pkg/twitter"

type ActivityResult struct {
	Tweet  *twitter.Tweet
	Action string
	Err    error
}
