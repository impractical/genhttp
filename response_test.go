package genhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adjust/goautoneg"
	"github.com/google/go-cmp/cmp"

	"impractical.co/apidiags"
)

func TestResponse(t *testing.T) {
	t.Parallel()

	rf := ResponseFactory{
		t: t,
	}
	resp := rf.NewResponse(context.Background(), httptest.NewRequest("GET", "/", nil))

	if _, ok := resp.enc.(jsonEncoder); !ok {
		t.Errorf("expected json encoder, got %T", resp.enc)
	}
}

type ResponseFactory struct {
	encoders []encoder
	t        *testing.T
}

// NewResponse returns a Response that is ready to be used, priming it to
// encode data in a format that matches the Accept header of the request.
func (rf ResponseFactory) NewResponse(_ context.Context, r *http.Request) *Response {
	var resp Response
	alts := make([]string, 0, len(rf.encoders))
	encTypes := map[string]encoder{}
	for _, enc := range rf.encoders {
		alts = append(alts, enc.contentType())
		encTypes[enc.contentType()] = enc
	}
	resp.enc = encTypes[goautoneg.Negotiate(r.Header.Get("Accept"), alts)]
	if resp.enc == nil {
		resp.enc = jsonEncoder{}
	}
	resp.t = rf.t
	return &resp
}

// Response represents information that should be conveyed back to the client.
type Response struct {
	t      *testing.T
	enc    encoder
	status int

	Diags []apidiags.Diagnostic `json:"diags,omitempty"`
}

// Equal returns true if two Responses should be considered equal. It is
// largely used to make testing Responses using go-cmp easier.
func (r Response) Equal(other Response) bool {
	if r.enc != other.enc {
		return false
	}
	if r.status != other.status {
		return false
	}
	if !cmp.Equal(r.Diags, other.Diags) {
		return false
	}
	return true
}

// HasErrors returns true if the Response has any error level diagnostics.
func (r *Response) HasErrors() bool {
	for _, diag := range r.Diags {
		if diag.Severity == apidiags.DiagnosticError {
			return true
		}
	}
	return false
}

// Send encodes the Response in a format that matches the Accept header, if at
// all possible, and writes it to the passed http.ResponseWriter.
func (r *Response) Send(_ context.Context, w http.ResponseWriter) {
	body, err := r.enc.encode(*r)
	if err != nil {
		r.t.Errorf("An unexpected error occurred sending the response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", r.enc.contentType())
	w.WriteHeader(r.status)
	n, err := w.Write(body)
	if err != nil {
		return
	}
	if n != len(body) {
		return
	}
}

// SetStatus sets the HTTP status code of the Response. It cannot be called
// after Response.Send.
func (r *Response) SetStatus(status int) {
	r.status = status
}

// AddError appends an error-level diagnostic to the Response. It cannot be
// called after Response.Send.
func (r *Response) AddError(code apidiags.Code, paths ...apidiags.Steps) {
	r.Diags = append(r.Diags, apidiags.Diagnostic{
		Severity: apidiags.DiagnosticError,
		Code:     code,
		Paths:    paths,
	})
}

// AddWarning appends a warning-level diagnostic to the Response. It cannot be
// called after Response.Send.
func (r *Response) AddWarning(code apidiags.Code, paths ...apidiags.Steps) {
	r.Diags = append(r.Diags, apidiags.Diagnostic{
		Severity: apidiags.DiagnosticWarning,
		Code:     code,
		Paths:    paths,
	})
}

// HandlePanic updates the Response in the face of a panic while
// processing the request.
func (r *Response) HandlePanic(_ context.Context, recoverArg any) {
	r.t.Fatalf("panic: %v", recoverArg)
}

// An encoder is a strategy for converting a Response into bytes in response to
// an Accept header.
type encoder interface {
	encode(Response) ([]byte, error)
	contentType() string
}
