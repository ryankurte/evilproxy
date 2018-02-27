package core

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/ryankurte/evilproxy/lib/plugins"
)

// Proxy core object
type Proxy struct {
	options Options
	backend Backend
	plugins plugins.PluginManager
}

// Backend interface for underlying request implementations
type Backend interface {
	Request(ctx interface{}, req *http.Request) (*http.Response, error)
}

// NewProxy creates a new proxy with the provided options
func NewProxy(options Options) *Proxy {
	p := Proxy{
		options: options,
		plugins: plugins.PluginManager{},
	}

	return &p
}

// BindBackend binds a backend interface for making real requests
func (p *Proxy) BindBackend(b Backend) {
	p.backend = b
}

// BindPlugin binds a plugin for processing requests and/or responses
func (p *Proxy) BindPlugin(h interface{}) {
	p.plugins.Bind(h)
}

// HandleRequest routes a request through the proxy and returns a response
func (p *Proxy) HandleRequest(req *http.Request) (*http.Response, error) {

	ctx := int(0)

	// Process request object
	if req.Body == nil {
		req.Header, _ = p.plugins.ProcessRequest(ctx, req.Header, "")
	} else {
		reqBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Printf("Error loading request body: %s", err)
		} else {
			// Process request
			reqHeader, reqBody := p.plugins.ProcessRequest(ctx, req.Header, string(reqBody))
			req.Header = reqHeader
			req.Body = ioutil.NopCloser(bytes.NewReader([]byte(reqBody)))
		}
	}

	// Call underlying proxy backend
	resp, err := p.backend.Request(ctx, req)
	if err != nil {
		log.Printf("Error making backend request %s", err)
		return nil, err
	}

	// Process response object
	if resp.Body == nil {
		resp.Header, _ = p.plugins.ProcessResponse(ctx, resp.Header, "")
	} else {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error loading response body: %s", err)
		} else {
			respHeader, respBody := p.plugins.ProcessResponse(ctx, resp.Header, string(respBody))
			resp.Header = respHeader
			resp.Body = ioutil.NopCloser(bytes.NewReader([]byte(respBody)))
		}
	}

	return resp, nil
}
