package core

import (
	"net/http"
)

// HTTPBackend implements a simple http client backend
type HTTPBackend struct {
}

// Request forwards the provided request and returns the response
func (b *HTTPBackend) Request(ctx interface{}, req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}
