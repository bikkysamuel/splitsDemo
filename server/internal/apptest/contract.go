package apptest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
)

// The contract every HTTP-seam response is checked against (doc 09).
var contract = sync.OnceValues(loadContract)

func loadContract() (routers.Router, error) {
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
	router, err := legacy.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("route %s: %w", specPath, err)
	}
	return router, nil
}

// checkContract fails the test if the request's route or the response's
// status, headers or body are not what api/openapi.yaml describes.
func checkContract(t testing.TB, req *http.Request, resp Response) {
	t.Helper()
	router, err := contract()
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
