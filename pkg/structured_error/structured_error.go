package structured_error

type ErrorType int

const (
	CannotSplitMessage ErrorType = iota
	RateLimited
	TwitterError
	DuplicateTweet
	TweetTooLong
	UserBlockedBot
	WrongMediaType
	NoPhotosFound
	OCRError
	DescribeError
	TranslateError
	UnsupportedLanguage
	Unknown
)

type StructuredError interface {
	error
	Type() ErrorType
}

type wrappedErr struct {
	err       error
	errorType ErrorType
}

func (e *wrappedErr) Error() string {
	return e.err.Error()
}

func (e *wrappedErr) Unwrap() error {
	return e.err
}

func (e *wrappedErr) Type() ErrorType {
	return e.errorType
}

func Wrap(err error, errorType ErrorType) StructuredError {
	if err == nil {
		return nil
	}
	if sErr, ok := err.(StructuredError); ok {
		return sErr
	}
	return &wrappedErr{err: err, errorType: errorType}
}
