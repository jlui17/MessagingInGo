package common

import "errors"

var (
	ErrNoUsername      = errors.New("no username")
	ErrInvalidUsername = errors.New("invalid username")
	ErrUsernameExists  = errors.New("username already exists")
)
