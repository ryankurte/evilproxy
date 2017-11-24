package ingress

import (
	"net/http"
)

// Proxy interface wrapped by ingress module
type Proxy interface {
	HandleRequest(*http.Request) (*http.Response, error)
}
