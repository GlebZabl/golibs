package server

import "golibs/errors"

var (
	BindingError = errors.NewWrapper("binding error", errors.ValidationErrorType)
)
