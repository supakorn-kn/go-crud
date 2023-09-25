package errors

const (
	UnknownErrorCode = 100_001
)

var UnknownError = new(UnknownErrorCode, "UnknownError", "unexpected error: %s")
