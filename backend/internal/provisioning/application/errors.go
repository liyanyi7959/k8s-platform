package application

import (
	"errors"
	"strings"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrInvalidParams = errors.New("invalid params")
)

type Error struct {
	Kind    error
	Message string
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Kind }

func (e *Error) ErrorCode() string {
	switch {
	case errors.Is(e.Kind, ErrInvalidParams):
		return "invalid_params"
	case errors.Is(e.Kind, ErrNotFound):
		return "not_found"
	case errors.Is(e.Kind, ErrConflict):
		return "conflict"
	default:
		return ""
	}
}

func (e *Error) UserMessage() string { return strings.TrimSpace(e.Message) }

func ErrWithMessage(kind error, message string) error {
	if kind == nil {
		return errors.New(strings.TrimSpace(message))
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return kind
	}
	return &Error{Kind: kind, Message: message}
}

type PageResult[T any] struct {
	List     []T `json:"list"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
