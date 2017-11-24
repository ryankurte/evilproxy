/**
 * Plugins package defines plugins for evilproxy
 * These should write/rewrite requests and responses for nefarious purposes
 *
 * Copyright 2017 Ryan Kurte
 */

package plugins

import (
	"net/http"
)

// RequestHeaderHandler interface implemented by plugins that re-write request headers
type RequestHeaderHandler interface {
	ProcessRequestHeader(ctx interface{}, req http.Header) http.Header
}

// RequestBodyHandler interface implemented by plugins that re-write the request body
type RequestBodyHandler interface {
	ProcessRequestBody(ctx interface{}, req string) string
}

// ResponseHeaderHandler interface implemented by plugins that re-writes the response header
type ResponseHeaderHandler interface {
	ProcessResponseHeader(ctx interface{}, resp http.Header) http.Header
}

// ResponseBodyHandler interface implemented by plugins that re-writes the response body
type ResponseBodyHandler interface {
	ProcessResponseBody(ctx interface{}, resp string) string
}

// PluginManager wraps plugin types and calls each sequentially when the appropriate method is called
type PluginManager struct {
	RequestHeaderHandlers  []RequestHeaderHandler
	RequestBodyHandlers    []RequestBodyHandler
	ResponseHeaderHandlers []ResponseHeaderHandler
	ResponseBodyHandlers   []ResponseBodyHandler
}

// Bind attaches a plugin to the PluginManager
func (pm *PluginManager) Bind(handler interface{}) {
	if r, ok := handler.(RequestHeaderHandler); ok {
		pm.RequestHeaderHandlers = append(pm.RequestHeaderHandlers, r)
	}
	if r, ok := handler.(RequestBodyHandler); ok {
		pm.RequestBodyHandlers = append(pm.RequestBodyHandlers, r)
	}
	if r, ok := handler.(ResponseHeaderHandler); ok {
		pm.ResponseHeaderHandlers = append(pm.ResponseHeaderHandlers, r)
	}
	if r, ok := handler.(ResponseBodyHandler); ok {
		pm.ResponseBodyHandlers = append(pm.ResponseBodyHandlers, r)
	}
}

// ProcessRequestHeader processes a request header through the bound plugins
func (pm *PluginManager) ProcessRequestHeader(ctx interface{}, req http.Header) http.Header {
	for _, h := range pm.RequestHeaderHandlers {
		req = h.ProcessRequestHeader(ctx, req)
	}
	return req
}

// ProcessRequestBody processes a request body through the bound plugins
func (pm *PluginManager) ProcessRequestBody(ctx interface{}, req string) string {
	for _, h := range pm.RequestBodyHandlers {
		req = h.ProcessRequestBody(ctx, req)
	}
	return req
}

// ProcessResponseHeader processes a response header through bound plugins
func (pm *PluginManager) ProcessResponseHeader(ctx interface{}, resp http.Header) http.Header {
	for _, h := range pm.ResponseHeaderHandlers {
		resp = h.ProcessResponseHeader(ctx, resp)
	}
	return resp
}

// ProcessResponseBody processes a response body through bound plugins
func (pm *PluginManager) ProcessResponseBody(ctx interface{}, resp string) string {
	for _, h := range pm.ResponseBodyHandlers {
		resp = h.ProcessResponseBody(ctx, resp)
	}
	return resp
}
