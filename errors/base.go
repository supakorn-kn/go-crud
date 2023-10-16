package errors

import (
	"fmt"
	"reflect"
)

type Error interface {
	error
	New(args ...any) BaseError
}

type ErrorType int

const (
	InternalServerError ErrorType = iota
	ResponseError
)

type BaseError struct {
	Code    int    `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`

	ErrorType     ErrorType `json:"-"`
	messageFormat string
}

func (e BaseError) Error() string {
	return e.Message
}

func (e *BaseError) New(args ...any) BaseError {

	if args == nil {
		e.Message = e.messageFormat
		return *e
	}

	e.Message = fmt.Sprintf(e.messageFormat, args)
	return *e
}

func (e BaseError) IsNil() bool {
	return reflect.ValueOf(e).IsZero()
}

func TryAssertError(err error) (BaseError, bool) {

	asserted, ok := err.(BaseError)
	return asserted, ok
}

func IsError(err error, expectedError BaseError) bool {

	asserted, ok := err.(BaseError)
	if !ok {
		return false
	}

	return asserted.Code == expectedError.Code && asserted.Message == expectedError.Message
}

func new(errorCode int, errorType ErrorType, name string, messageFormat string) Error {

	return &BaseError{Code: errorCode, ErrorType: errorType, Name: name, messageFormat: messageFormat}
}
