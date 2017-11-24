package core

import (
	"net/http"
)

type HTTPBackend struct {
}

func (b *HTTPBackend) Request(ctx interface{}, req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}
