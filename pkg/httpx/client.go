package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	baseURL      *url.URL
	httpClient   *http.Client
	headers      http.Header
	middlewares  []Middleware
	retryMax     int
	retryBackoff time.Duration
}

func New(opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Transport: defaultTransport(),
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) Get(ctx context.Context, path string, opts ...RequestOption) (*Response, error) {
	return c.request(ctx, http.MethodGet, path, nil, opts...)
}

func (c *Client) Post(ctx context.Context, path string, body any, opts ...RequestOption) (*Response, error) {
	return c.request(ctx, http.MethodPost, path, body, opts...)
}

func (c *Client) Put(ctx context.Context, path string, body any, opts ...RequestOption) (*Response, error) {
	return c.request(ctx, http.MethodPut, path, body, opts...)
}

func (c *Client) Delete(ctx context.Context, path string, opts ...RequestOption) (*Response, error) {
	return c.request(ctx, http.MethodDelete, path, nil, opts...)
}

func (c *Client) Patch(ctx context.Context, path string, body any, opts ...RequestOption) (*Response, error) {
	return c.request(ctx, http.MethodPatch, path, body, opts...)
}

func (c *Client) Head(ctx context.Context, path string, opts ...RequestOption) (*Response, error) {
	return c.request(ctx, http.MethodHead, path, nil, opts...)
}

func (c *Client) Options(ctx context.Context, path string, opts ...RequestOption) (*Response, error) {
	return c.request(ctx, http.MethodOptions, path, nil, opts...)
}

func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	return c.execute(ctx, req)
}

func (c *Client) request(ctx context.Context, method, path string, body any, opts ...RequestOption) (*Response, error) {
	req := &Request{
		Method: method,
		URL:    path,
		Body:   body,
	}
	for _, opt := range opts {
		opt(req)
	}
	return c.execute(ctx, req)
}

func (c *Client) execute(ctx context.Context, req *Request) (*Response, error) {
	u, err := c.resolveURL(req.URL, req.QueryParams)
	if err != nil {
		return nil, NewError(ErrorTypeParse, "invalid URL", err)
	}
	req.resolvedURL = u

	finalCtx := ctx
	if req.ctx != nil {
		finalCtx = req.ctx
	}
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		finalCtx, cancel = context.WithTimeout(finalCtx, req.Timeout)
		defer cancel()
	}
	req.resolvedCtx = finalCtx

	// Buffer io.Reader body so it can be re-read across retries
	if reader, ok := req.Body.(io.Reader); ok {
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, NewError(ErrorTypeParse, "read request body", err)
		}
		req.Body = data
	}

	// coreHandler builds http.Request from the (possibly middleware-modified) req
	coreHandler := Handler(func(r *Request) (*Response, error) {
		bodyReader, contentType, err := encodeBody(r.Body)
		if err != nil {
			return nil, NewError(ErrorTypeParse, "encode request body", err)
		}

		httpReq, err := http.NewRequestWithContext(r.resolvedCtx, r.Method, r.resolvedURL, bodyReader)
		if err != nil {
			return nil, NewError(ErrorTypeParse, "create request", err)
		}

		for k, vs := range c.headers {
			for _, v := range vs {
				httpReq.Header.Add(k, v)
			}
		}
		if contentType != "" && httpReq.Header.Get("Content-Type") == "" {
			httpReq.Header.Set("Content-Type", contentType)
		}
		for k, vs := range r.Header {
			for _, v := range vs {
				httpReq.Header.Set(k, v)
			}
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return nil, mapError(err)
		}
		return newResponse(resp)
	})

	handler := coreHandler

	if c.retryMax > 0 {
		handler = RetryMiddleware(c.retryMax, c.retryBackoff)(handler)
	}

	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}

	return handler(req)
}

func (c *Client) resolveURL(path string, query url.Values) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		u, err := url.Parse(path)
		if err != nil {
			return "", err
		}
		if len(query) > 0 {
			q := u.Query()
			for k, vs := range query {
				for _, v := range vs {
					q.Add(k, v)
				}
			}
			u.RawQuery = q.Encode()
		}
		return u.String(), nil
	}

	if c.baseURL == nil {
		return "", fmt.Errorf("httpx: no base URL configured; use WithBaseURL or pass an absolute URL")
	}

	rel, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	u := c.baseURL.ResolveReference(rel)
	if len(query) > 0 {
		q := u.Query()
		for k, vs := range query {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func encodeBody(body any) (io.Reader, string, error) {
	switch v := body.(type) {
	case nil:
		return nil, "", nil
	case string:
		return strings.NewReader(v), "text/plain; charset=utf-8", nil
	case []byte:
		return bytes.NewReader(v), "application/octet-stream", nil
	case url.Values:
		return encodeFormBody(v)
	case io.Reader:
		return v, "application/octet-stream", nil
	default:
		if fd, ok := body.(*FormData); ok {
			return encodeMultipartBody(fd)
		}
		data, err := json.Marshal(v)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(data), "application/json; charset=utf-8", nil
	}
}

func encodeFormBody(v url.Values) (io.Reader, string, error) {
	return strings.NewReader(v.Encode()), "application/x-www-form-urlencoded", nil
}

func encodeMultipartBody(fd *FormData) (io.Reader, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range fd.Fields {
		w.WriteField(f.key, f.value)
	}
	for _, f := range fd.files {
		if f.data != nil {
			part, err := w.CreateFormFile(f.key, f.filename)
			if err != nil {
				return nil, "", err
			}
			if _, err := io.Copy(part, bytes.NewReader(f.data)); err != nil {
				return nil, "", err
			}
			continue
		}
		part, err := w.CreateFormFile(f.key, filepath.Base(f.path))
		if err != nil {
			return nil, "", err
		}
		file, err := os.Open(f.path)
		if err != nil {
			return nil, "", err
		}
		if _, err := io.Copy(part, file); err != nil {
			file.Close()
			return nil, "", err
		}
		file.Close()
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}
