package errors

import (
	standart "errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	DoesNotExistErrorType = "DOES_NOT_EXIST_ERROR"
	AlreadyExistErrorType = "ALREADY_EXIST_ERROR"
	InconsistentErrorType = "INCONSISTENT_ERROR"
	ValidationErrorType   = "VALIDATION_ERROR"
	GeneralErrorType      = "GENERAL_ERROR"
	ForbiddenErrorType    = "FORBIDDEN_ERROR"
	BusinessErrorType     = "BUSINESS_ERROR"
)

type wrapped interface {
	Origin() error
}

type Causer interface {
	Cause() error
}

type HasType interface {
	Type() string
}

func IsType(err error, types ...string) bool {
	if typer, ok := err.(HasType); ok {
		for _, typ := range types {
			if typer.Type() == typ {
				return true
			}
		}
	}
	if wrapped, ok := err.(wrapped); ok {
		return IsType(wrapped.Origin(), types...)
	}

	return false
}

type baseError struct {
	origin error
	cause  error
	typ    string
	stack  []string
	code   string
}

func (b baseError) WithCode(code string) *baseError {
	b.code = code
	if coder, ok := b.origin.(interface{ WithCode(string) *baseError }); ok {
		b.origin = coder.WithCode(code)
	}

	return &b
}

func (b *baseError) Code() string {
	if base, ok := b.origin.(*baseError); ok {
		return base.Code()
	}

	if b.code == "" {
		return b.typ
	}
	return b.code
}

func (b *baseError) Type() string {
	return b.typ
}

func (b *baseError) Origin() error {
	return b.origin
}

func (b *baseError) Error() string {
	switch b.origin.(type) {
	case *baseError:
		return b.origin.Error()
	default:
		stack := b.getStack()
		var originError string
		if b.origin != nil {
			originError = b.origin.Error()
		} else {
			originError = b.cause.Error()
		}
		return fmt.Sprintf("%s %s", stack, originError)
	}
}

func (b *baseError) getStack() string {
	result := "\n"
	space := ""
	for lvl := len(b.stack) - 1; lvl >= 0; lvl-- {
		result += fmt.Sprintf("%s%s\n", space, b.stack[lvl])
		space += " "
	}
	return result + space
}

func (b *baseError) Cause() error {
	return b.cause
}

type wrapper struct {
	msg string
	typ string
}

func (w *wrapper) Error() string {
	return w.msg
}

// Be careful using lvl, incorrect value may cause panic!!!
func (w *wrapper) Wrap(err error, lvl ...int) *baseError {
	level := 0
	if len(lvl) == 1 {
		level = lvl[0]
	}

	return &baseError{
		origin: err,
		cause:  w,
		typ:    w.typ,
		stack:  getStackTrace(level),
	}
}

func (w *wrapper) New(msg string) *baseError {
	return &baseError{
		origin: standart.New(msg),
		cause:  w,
		typ:    w.typ,
		stack:  getStackTrace(0),
	}
}

func (w *wrapper) NewF(format string, args ...interface{}) error {
	return w.New(fmt.Sprintf(format, args...))
}

func New(msg string) error {
	err := errors.New(msg)
	return &baseError{
		origin: err,
		cause:  nil,
		stack:  getStackTrace(0),
	}
}

func NewWrapper(msg string, typ ...string) *wrapper {
	Type := ""
	if len(typ) == 1 {
		Type = typ[0]
	}
	return &wrapper{
		msg,
		Type,
	}
}

func IsCausedBy(err error, reasons ...error) bool {
	for _, reason := range reasons {
		if errors.Cause(err) == reason {
			return true
		}
	}
	return false
}

func GetCode(err error) string {
	switch err.(type) {
	case interface{ Code() string }:
		return err.(interface{ Code() string }).Code()
	default:
		return GeneralErrorType
	}
}

func GetInsideErrMsg(err error) string {
	switch err.(type) {
	case *baseError:
		if err.(*baseError).origin != nil {
			return GetInsideErrMsg(err.(*baseError).origin)
		} else {
			return err.Error()
		}
	default:
		return err.Error()
	}
}

func getStackTrace(lvl int) []string {
	var result []string
	lvl += 2
	for {
		pc, _, line, _ := runtime.Caller(lvl)
		file, line := runtime.FuncForPC(pc).FileLine(pc)
		funcName := runtime.FuncForPC(pc).Name()
		result = append(result, file+":"+strconv.Itoa(line))
		if strings.Contains(funcName, "Main") || strings.Contains(funcName, "gin.(*Context).Next") || strings.Contains(funcName, "Test") || strings.Contains(funcName, "runtime.goexit") {
			return result
		}
		lvl++
	}
}
