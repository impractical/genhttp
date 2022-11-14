package genhttp

import (
	"context"
	"net/http"
	"testing"
)

var _ Responder = &Response{}

type testRequest struct{}

type testHandler struct{}

func (testHandler) ParseRequest(ctx context.Context, _ *http.Request, _ *Response) (testRequest, context.Context) {
	return testRequest{}, ctx
}

func (testHandler) ValidateRequest(ctx context.Context, _ testRequest, _ *Response) context.Context {
	return ctx
}

func (testHandler) ExecuteRequest(ctx context.Context, _ testRequest, _ *Response) context.Context {
	return ctx
}

var _ Handler[testRequest, *Response] = testHandler{}

func TestHandler(t *testing.T) {
	t.Parallel()

	factory := ResponseFactory{
		encoders: []encoder{
			jsonEncoder{},
		},
	}

	Handle[testRequest, *Response](factory, testHandler{})
}
