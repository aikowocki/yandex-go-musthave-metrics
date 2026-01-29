package httpError

import "net/http"

type HTTPError struct {
	Code    int
	Message string
}

func (e *HTTPError) Error() string { return e.Message }
func (e *HTTPError) Status() int   { return e.Code }

func NewHTTPError(code int, msg string) *HTTPError {
	return &HTTPError{Code: code, Message: msg}
}

func NotFound(msg string) *HTTPError {
	if msg == "" {
		msg = http.StatusText(http.StatusNotFound)
	}
	return NewHTTPError(http.StatusNotFound, msg)
}

func BadRequest(msg string) *HTTPError {
	if msg == "" {
		msg = http.StatusText(http.StatusBadRequest)
	}
	return NewHTTPError(http.StatusBadRequest, msg)
}
