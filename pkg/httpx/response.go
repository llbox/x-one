package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Response struct {
	*http.Response
	body []byte
}

func newResponse(resp *http.Response) (*Response, error) {
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, NewError(ErrorTypeParse, "read response body", err)
	}
	return &Response{
		Response: resp,
		body:     body,
	}, nil
}

func (r *Response) BodyBytes() []byte {
	return r.body
}

func (r *Response) BodyString() string {
	return string(r.body)
}

func (r *Response) JSON(v any) error {
	if err := json.Unmarshal(r.body, v); err != nil {
		return NewError(ErrorTypeParse, "decode JSON response", err)
	}
	return nil
}

func (r *Response) Status() int {
	return r.StatusCode
}

func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

func (r *Response) IsError() bool {
	return r.StatusCode >= 400
}

func (r *Response) String() string {
	return fmt.Sprintf("%d %s", r.StatusCode, http.StatusText(r.StatusCode))
}
