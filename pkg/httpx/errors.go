package httpx

import (
	"context"
	"errors"
	"fmt"
	"net"
)

type ErrorType string

const (
	ErrorTypeNetwork ErrorType = "network"
	ErrorTypeTimeout ErrorType = "timeout"
	ErrorTypeHTTP    ErrorType = "http"
	ErrorTypeParse   ErrorType = "parse"
	ErrorTypeContext ErrorType = "context"
)

type ClientError struct {
	Type       ErrorType
	Message    string
	Err        error
	StatusCode int
}

func (e *ClientError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *ClientError) Unwrap() error {
	return e.Err
}

func (e *ClientError) WithStatus(code int) *ClientError {
	e.StatusCode = code
	return e
}

func NewError(errType ErrorType, msg string, err error) *ClientError {
	return &ClientError{
		Type:    errType,
		Message: msg,
		Err:     err,
	}
}

func mapError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return NewError(ErrorTypeContext, "context error", err)
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return NewError(ErrorTypeTimeout, "request timeout", err)
		}
		return NewError(ErrorTypeNetwork, "network error", err)
	}
	return NewError(ErrorTypeNetwork, "request failed", err)
}

func IsNetworkError(err error) bool {
	var ce *ClientError
	return errors.As(err, &ce) && ce.Type == ErrorTypeNetwork
}

func IsTimeout(err error) bool {
	var ce *ClientError
	return errors.As(err, &ce) && ce.Type == ErrorTypeTimeout
}

func IsHTTPError(err error) bool {
	var ce *ClientError
	return errors.As(err, &ce) && ce.Type == ErrorTypeHTTP
}

func IsStatus(err error, code int) bool {
	var ce *ClientError
	return errors.As(err, &ce) && ce.StatusCode == code
}

func IsRetryable(err error) bool {
	var ce *ClientError
	if errors.As(err, &ce) {
		return ce.Type == ErrorTypeNetwork || ce.Type == ErrorTypeTimeout
	}
	return false
}
