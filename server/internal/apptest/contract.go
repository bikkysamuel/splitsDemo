package apptest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
)

// contractRouter matches requests to api/openapi.yaml operations so every
// HTTP-seam response can be checked against the contract (doc 09).
var contractRouter = sync.OnceValues(loadContractRouter)

// contractDoc is api/openapi.yaml, loaded and validated once.
var contractDoc = sync.OnceValues(func() (*openapi3.T, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("locate apptest source file")
	}
	specPath := filepath.Join(filepath.Dir(file), "..", "..", "..", "api", "openapi.yaml")

	doc, err := openapi3.NewLoader().LoadFromFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", specPath, err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate %s: %w", specPath, err)
	}
	// Match routes by path alone: test servers listen on random ports.
	doc.Servers = nil
	return doc, nil
})

func loadContractRouter() (routers.Router, error) {
	doc, err := contractDoc()
	if err != nil {
		return nil, err
	}
	router, err := legacy.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("route api/openapi.yaml: %w", err)
	}
	return router, nil
}

// checkContract fails the test if the request's route or the response's
// status, headers or body are not what api/openapi.yaml describes.
func checkContract(t testing.TB, req *http.Request, resp Response) {
	t.Helper()
	router, err := contractRouter()
	if err != nil {
		t.Fatalf("contract: %v", err)
	}
	route, pathParams, err := router.FindRoute(req)
	if err != nil {
		t.Errorf("%s %s is not in api/openapi.yaml: %v", req.Method, req.URL.Path, err)
		return
	}
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		},
		Status: resp.StatusCode,
		Header: resp.Header,
		Body:   io.NopCloser(bytes.NewReader(resp.Body)),
		Options: &openapi3filter.Options{
			IncludeResponseStatus: true,
			MultiError:            true,
		},
	}
	if err := openapi3filter.ValidateResponse(req.Context(), input); err != nil {
		t.Errorf("%s %s → %d breaks api/openapi.yaml:\n%v\nbody: %s",
			req.Method, req.URL.Path, resp.StatusCode, err, resp.Body)
	}
}

// Operation is one operation of api/openapi.yaml.
type Operation struct {
	Method string
	// Path has every path parameter filled with a fixed UUID.
	Path string
	// BearerAuth reports whether the operation requires an access token.
	BearerAuth bool
	// IdempotencyKey reports whether the operation declares the
	// Idempotency-Key header.
	IdempotencyKey bool
	// ReadOnly reports `x-splits-read-only: true`: a POST that changes
	// nothing.
	ReadOnly bool
}

// Operations lists every operation in api/openapi.yaml, so tests can sweep
// rules that hold for all of them.
func Operations(t testing.TB) []Operation {
	t.Helper()
	doc, err := contractDoc()
	if err != nil {
		t.Fatalf("contract: %v", err)
	}
	var ops []Operation
	for path, item := range doc.Paths.Map() {
		filled := pathParam.ReplaceAllString(path, "0190b6c4-0000-7000-8000-000000000001")
		for method, op := range item.Operations() {
			security := doc.Security
			if op.Security != nil {
				security = *op.Security
			}
			o := Operation{Method: method, Path: filled, ReadOnly: op.Extensions["x-splits-read-only"] == true}
			for _, req := range security {
				if _, ok := req["bearerAuth"]; ok {
					o.BearerAuth = true
				}
			}
			for _, p := range op.Parameters {
				if p.Value != nil && p.Value.In == openapi3.ParameterInHeader && p.Value.Name == "Idempotency-Key" {
					o.IdempotencyKey = true
				}
			}
			ops = append(ops, o)
		}
	}
	return ops
}

var pathParam = regexp.MustCompile(`\{[^}]+\}`)
