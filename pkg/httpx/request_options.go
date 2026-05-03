package httpx

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type Request struct {
	Method      string
	URL         string
	Header      http.Header
	Body        any
	QueryParams url.Values
	Timeout     time.Duration
	ctx         context.Context

	resolvedURL string
	resolvedCtx context.Context
}

func (r *Request) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

type RequestOption func(*Request)

func WithQuery(key, value string) RequestOption {
	return func(r *Request) {
		if r.QueryParams == nil {
			r.QueryParams = url.Values{}
		}
		r.QueryParams.Add(key, value)
	}
}

func WithQueryParams(params map[string]string) RequestOption {
	return func(r *Request) {
		if r.QueryParams == nil {
			r.QueryParams = url.Values{}
		}
		for k, v := range params {
			r.QueryParams.Set(k, v)
		}
	}
}

func WithHeader(key, value string) RequestOption {
	return func(r *Request) {
		if r.Header == nil {
			r.Header = http.Header{}
		}
		r.Header.Set(key, value)
	}
}

func WithRequestTimeout(d time.Duration) RequestOption {
	return func(r *Request) {
		r.Timeout = d
	}
}

func WithContext(ctx context.Context) RequestOption {
	return func(r *Request) {
		r.ctx = ctx
	}
}
