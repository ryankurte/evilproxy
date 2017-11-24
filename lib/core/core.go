package core

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/ryankurte/experiments/evilproxy/lib/plugins"
)

// Proxy core object
type Proxy struct {
	options Options
	backend Backend
	plugins plugins.PluginManager
}

type Backend interface {
	Request(ctx interface{}, req *http.Request) (*http.Response, error)
}

type ByteReadCloser struct {
	io.Reader
}

func (b *ByteReadCloser) Close() error { return nil }

// NewProxy creates a new proxy with the provided options
func NewProxy(options Options) *Proxy {
	p := Proxy{
		options: options,
		plugins: plugins.PluginManager{},
	}
	return &p
}

func (p *Proxy) BindBackend(b Backend) {
	p.backend = b
}

// HandleRequest routes a request through the proxy and returns a response
func (p *Proxy) HandleRequest(req *http.Request) (*http.Response, error) {

	ctx := int(0)

	// Process request headers
	req.Header = p.plugins.ProcessRequestHeader(ctx, req.Header)

	// Process request body if available
	if req.Body != nil {
		requestBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Printf("Error loading request body: %s", err)
		} else {
			requestBody := p.plugins.ProcessRequestBody(ctx, string(requestBody))
			req.Body = &ByteReadCloser{bytes.NewReader([]byte(requestBody))}
		}
	}

	// Call underlying proxy backend
	resp, err := p.backend.Request(ctx, req)
	if err != nil {
		log.Printf("Error making backend request %s", err)
		return nil, err
	}

	// Process response headers
	resp.Header = p.plugins.ProcessResponseHeader(ctx, resp.Header)

	// Process response body if available
	if resp.Body != nil {
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error loading response body: %s", err)
		} else {
			responseBody := p.plugins.ProcessResponseBody(ctx, string(responseBody))
			resp.Body = &ByteReadCloser{bytes.NewReader([]byte(responseBody))}
		}
	}

	return resp, nil
}
