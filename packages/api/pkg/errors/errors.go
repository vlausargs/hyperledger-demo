package errors

type AppError struct {
	Code    int
	Message string
	Detail  string
}

// Error satisfies the error interface so *AppError can be returned as error
// from constructors that wrap external libraries. Required for type contract
// even when callers handle *AppError concretely.
func (e *AppError) Error() string {
	return e.Message
}

func NewBadRequest(msg string) *AppError {
	return &AppError{Code: 400, Message: msg}
}

func NewNotFound(msg string) *AppError {
	return &AppError{Code: 404, Message: msg}
}

func NewConflict(msg string) *AppError {
	return &AppError{Code: 409, Message: msg}
}

func NewInternal(msg, detail string) *AppError {
	return &AppError{Code: 500, Message: msg, Detail: detail}
}

func NewUnauthorized(msg string) *AppError {
	return &AppError{Code: 401, Message: msg}
}

func NewForbidden(msg string) *AppError {
	return &AppError{Code: 403, Message: msg}
}

func NewUnavailable(msg string) *AppError {
	return &AppError{Code: 503, Message: msg}
}
