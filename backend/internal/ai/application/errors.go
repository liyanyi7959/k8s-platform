package application

import (
	"errors"
	"strings"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrConflict            = errors.New("conflict")
	ErrInvalidParams       = errors.New("invalid params")
	ErrCrypto              = errors.New("crypto error")
	ErrProviderUnavailable = errors.New("provider unavailable")
)

// Error keeps an application error category and an API-safe message together.
// The legacy HTTP error writer recognizes the small ErrorCode/UserMessage
// contract without the application layer importing a legacy package.
type Error struct {
	Kind    error
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Kind
}

func (e *Error) ErrorCode() string {
	switch {
	case errors.Is(e.Kind, ErrInvalidParams):
		return "invalid_params"
	case errors.Is(e.Kind, ErrNotFound):
		return "not_found"
	case errors.Is(e.Kind, ErrConflict):
		return "conflict"
	case errors.Is(e.Kind, ErrCrypto):
		return "crypto"
	case errors.Is(e.Kind, ErrProviderUnavailable):
		return "provider_unavailable"
	default:
		return ""
	}
}

func (e *Error) UserMessage() string {
	return strings.TrimSpace(e.Message)
}

func ErrorWithMessage(kind error, message string) error {
	if kind == nil {
		return errors.New(strings.TrimSpace(message))
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return kind
	}
	return &Error{Kind: kind, Message: message}
}

// ErrWithMessage keeps the familiar service-layer call shape while the AI
// use cases are consumed directly from the application package.
func ErrWithMessage(kind error, message string) error {
	return ErrorWithMessage(kind, message)
}
