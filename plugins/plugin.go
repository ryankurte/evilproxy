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
	ProcessRequestHeader(req *http.Header) *http.Header
}

// RequestBodyHandler interface implemented by plugins that re-write the request body
type RequestBodyHandler interface {
	ProcessRequestBody(req []byte) []byte
}

// ResponseHandler interface implemented by plugins that re-writes the response body
type ResponseHeaderHandler interface {
	ProcessResponseHeader(*http.Header) *http.Header
}

// ResponseHandler interface implemented by plugins that re-writes the response body
type ResponseBodyHandler interface {
	ProcessResponseBody([]byte) []byte
}

type PluginManager struct {
	RequestHeaderHandlers  []RequestHeaderHandler
	RequestBodyHandlers    []RequestBodyHandler
	ResponseHeaderHandlers []ResponseHeaderHandler
	ResponseBodyHandlers   []ResponseBodyHandler
}

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

func (pm *PluginManager) ProcessRequestHeader(req *http.Header) *http.Header {
	for _, h := range pm.RequestHeaderHandlers {
		req = h.ProcessRequestHeader(req)
	}
	return req
}

func (pm *PluginManager) ProcessRequestBody(req []byte) []byte {
	for _, h := range pm.RequestBodyHandlers {
		req = h.ProcessRequestBody(req)
	}
	return req
}

func (pm *PluginManager) ProcessResponseHeader(resp *http.Header) *http.Header {
	for _, h := range pm.ResponseHeaderHandlers {
		resp = h.ProcessResponseHeader(resp)
	}
	return resp
}

func (pm *PluginManager) ProcessResponseBody(resp []byte) []byte {
	for _, h := range pm.ResponseBodyHandlers {
		resp = h.ProcessResponseBody(resp)
	}
	return resp
}
