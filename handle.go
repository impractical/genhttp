package genhttp

import (
	"context"
	"net/http"
)

// ResponseCreator is a factory type that is capable of generating new
// Responder instances that are ready to be used.
//
// We use a type parameter here instead of using the interface directly because
// we want the response types to be able to be structs with encoding tags or
// whatever, and to have their fields be directly accessible without a type
// assertion, which the Responder interface isn't able to provide.
type ResponseCreator[Response Responder] interface {
	NewResponse(context.Context, *http.Request) Response
}

// Responder is an interface describing a response type.
type Responder interface {
	// HasErrors should return true if the response is considered an error
	// response and processing should not proceed.
	HasErrors() bool

	// Send writes the response to the http.ResponseWriter.
	Send(context.Context, http.ResponseWriter)

	// HandlePanic updates the Response in the face of a panic while
	// processing the request.
	HandlePanic(ctx context.Context, recoverArg any)
}

// Redirecter is an interface describing a Responder that sometimes responds by
// redirecting the client.
type Redirecter interface {
	// RedirectTo returns the URL to redirect to and the HTTP status code
	// to use when redirecting. If the status code returned is between 300
	// and 399, inclusive, genhttp will call http.Redirect with the
	// returned URL and status code instead of calling Send.
	RedirectTo() (url string, status int)
}

// CookieWriter is an interface describing a Responder that sometimes responds
// by writing cookies to the client.
type CookieWriter interface {
	// WriteCookies returns the cookies to write. An attempt is always made
	// to write cookies.
	WriteCookies() []*http.Cookie
}

// Handler is an endpoint that is going to parse, validate, and execute an HTTP
// request. Its Request type parameter should be a type that can describe the
// request, usually a struct with JSON tags or something similar. The Response
// type parameter should be an implementation of the Responder interface,
// usually a pointer that can be modified in the ParseRequest, ValidateRequest,
// and ExecuteRequest methods.
type Handler[Request any, Response Responder] interface {
	// ParseRequest turns an `http.Request` into the Request type passed
	// in, usually by parsing some encoding. The returned context.Context
	// will be used as the new request context; if in doubt, return the
	// context.Context passed as an argument.
	ParseRequest(context.Context, *http.Request, Response) (Request, context.Context)

	// ValidateRequest checks that the passed Request is valid, for
	// whatever definition of valid suits the endpoint. The returned
	// context.Context will be used as the new request context; if in
	// doubt, return the context.Context passed as an argument.
	ValidateRequest(context.Context, Request, Response) context.Context

	// ExecuteRequest performs the action described by the Request. It can
	// assume that the Request is valid. The returned context.Context will
	// be used as the new request context; if in doubt, return the
	// context.Context passed as an argument.
	ExecuteRequest(context.Context, Request, Response) context.Context
}

// Handle provides an http.Handler that will call the passed Handler. `rf` is
// used to create a new instance of the Response type. Then the `ParseRequest`
// method of `h` will be called, followed by `ValidateRequest` and
// `ExecuteRequest`. At the end, the Response has its `Send` method called.
//
// If at any point in this process the Response's `HasErrors` method returns
// `true`, the Response's `Send` method is called and the function returns.
func Handle[Request any, Response Responder](rf ResponseCreator[Response], handler Handler[Request, Response]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		resp := rf.NewResponse(ctx, r)
		defer func() {
			if cw, ok := any(resp).(CookieWriter); ok {
				for _, cookie := range cw.WriteCookies() {
					http.SetCookie(w, cookie)
				}
			}
			if red, ok := any(resp).(Redirecter); ok {
				url, code := red.RedirectTo()
				if code >= 300 && code < 400 {
					http.Redirect(w, r, url, code)
					return
				}
			}
			resp.Send(ctx, w)
		}()
		defer func() {
			msg := recover()
			if msg == nil {
				return
			}
			resp.HandlePanic(ctx, msg)
		}()
		if resp.HasErrors() {
			return
		}

		var req Request
		req, ctx = handler.ParseRequest(ctx, r, resp)
		if resp.HasErrors() {
			return
		}

		ctx = handler.ValidateRequest(ctx, req, resp)
		if resp.HasErrors() {
			return
		}

		ctx = handler.ExecuteRequest(ctx, req, resp)
	})
}
