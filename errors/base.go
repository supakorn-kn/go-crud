package errors

import (
	"fmt"
	"reflect"
)

type Error interface {
	error
	New(args ...any) BaseError
}

type BaseError struct {
	Code    int    `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`

	messageFormat string
}

func (e BaseError) Error() string {
	return e.Message
}

func (e *BaseError) New(args ...any) BaseError {

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

func new(errorCode int, name string, messageFormat string) Error {

	return &BaseError{Code: errorCode, Name: name, messageFormat: messageFormat}
}
