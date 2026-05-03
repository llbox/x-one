package httpx

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

type ClientOption func(*Client)

func WithBaseURL(rawURL string) ClientOption {
	return func(c *Client) {
		u, err := url.Parse(rawURL)
		if err != nil {
			return
		}
		c.baseURL = u
	}
}

func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

func WithDefaultHeader(key, value string) ClientOption {
	return func(c *Client) {
		if c.headers == nil {
			c.headers = http.Header{}
		}
		c.headers.Set(key, value)
	}
}

func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

func WithUserAgent(ua string) ClientOption {
	return WithDefaultHeader("User-Agent", ua)
}

func WithMiddleware(mw ...Middleware) ClientOption {
	return func(c *Client) {
		c.middlewares = append(c.middlewares, mw...)
	}
}

func WithRetry(maxRetries int, backoff time.Duration) ClientOption {
	return func(c *Client) {
		c.retryMax = maxRetries
		c.retryBackoff = backoff
	}
}

func WithDebug(w io.Writer) ClientOption {
	return func(c *Client) {
		c.middlewares = append(c.middlewares, LoggingMiddleware(w))
	}
}
