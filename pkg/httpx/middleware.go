package httpx

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type Handler func(req *Request) (*Response, error)

type Middleware func(next Handler) Handler

func RetryMiddleware(maxRetries int, backoff time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(req *Request) (*Response, error) {
			var lastErr error
			for attempt := 0; attempt <= maxRetries; attempt++ {
				if attempt > 0 {
					time.Sleep(backoff * time.Duration(1<<uint(attempt-1)))
				}
				resp, err := next(req)
				if err == nil {
					if resp.IsSuccess() {
						return resp, nil
					}
					if resp.StatusCode < 500 {
						return resp, nil
					}
					lastErr = NewError(ErrorTypeHTTP, fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
					continue
				}
				if IsRetryable(err) {
					lastErr = err
					continue
				}
				return nil, err
			}
			return nil, lastErr
		}
	}
}

func LoggingMiddleware(w io.Writer) Middleware {
	return func(next Handler) Handler {
		return func(req *Request) (*Response, error) {
			start := time.Now()
			fmt.Fprintf(w, "[httpx] --> %s %s\n", req.Method, req.URL)

			resp, err := next(req)

			elapsed := time.Since(start)
			if err != nil {
				fmt.Fprintf(w, "[httpx] <-- ERROR %s (%s)\n", err, elapsed)
			} else {
				fmt.Fprintf(w, "[httpx] <-- %d %s (%s)\n", resp.StatusCode, http.StatusText(resp.StatusCode), elapsed)
			}
			return resp, err
		}
	}
}

func AuthMiddleware(scheme, value string) Middleware {
	return func(next Handler) Handler {
		return func(req *Request) (*Response, error) {
			if req.Header == nil {
				req.Header = http.Header{}
			}
			req.Header.Set("Authorization", scheme+" "+value)
			return next(req)
		}
	}
}
