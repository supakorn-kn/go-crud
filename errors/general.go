package errors

const (
	UnknownErrorCode = 100_001
)

// UnknownError indicates internal error when server catch unexpected or unregistered error while operating
var UnknownError = new(UnknownErrorCode, InternalServerError, "UnknownError", "unexpected error: %s")
