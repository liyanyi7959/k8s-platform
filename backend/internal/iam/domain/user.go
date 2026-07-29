package domain

import "errors"

var (
	ErrInvalidParams        = errors.New("invalid iam parameters")
	ErrUserNotFound         = errors.New("auth user not found")
	ErrUserDisabled         = errors.New("auth user disabled")
	ErrPasswordIncorrect    = errors.New("auth password incorrect")
	ErrOldPasswordIncorrect = errors.New("auth old password incorrect")
	ErrNotFound             = errors.New("iam resource not found")
	ErrConflict             = errors.New("iam resource conflict")
	ErrCaptchaNotFound      = errors.New("captcha not found or expired")
	ErrCaptchaInvalid       = errors.New("captcha verification failed")
	ErrCaptchaUsed          = errors.New("captcha already used")
	ErrResetTokenNotFound   = errors.New("reset token not found or expired")
	ErrResetTokenUsed       = errors.New("reset token already used")
)

type User struct {
	ID           uint64
	Username     string
	Email        string
	Status       string
	PasswordHash string
}
type Principal struct {
	ID          int64
	Username    string
	Status      string
	Roles       []string
	Permissions []string
}
