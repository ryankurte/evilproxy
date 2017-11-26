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

// RequestHandler interface implemented by plugins to re-write requests
type RequestHandler interface {
	ProcessRequest(ctx interface{}, header http.Header, body string) (http.Header, string)
}

// ResponseHandler interface implemented by plugins to re-write responses
type ResponseHandler interface {
	ProcessResponse(ctx interface{}, header http.Header, body string) (http.Header, string)
}

// PluginManager wraps plugin types and calls each sequentially when the appropriate method is called
type PluginManager struct {
	RequestHandlers  []RequestHandler
	ResponseHandlers []ResponseHandler
}

// Bind attaches a plugin to the PluginManager
func (pm *PluginManager) Bind(handler interface{}) {
	if r, ok := handler.(RequestHandler); ok {
		pm.RequestHandlers = append(pm.RequestHandlers, r)
	}
	if r, ok := handler.(ResponseHandler); ok {
		pm.ResponseHandlers = append(pm.ResponseHandlers, r)
	}
}

// ProcessRequest processes a request header through the bound plugins
func (pm *PluginManager) ProcessRequest(ctx interface{}, header http.Header, body string) (http.Header, string) {
	for _, h := range pm.RequestHandlers {
		header, body = h.ProcessRequest(ctx, header, body)
	}
	return header, body
}

// ProcessResponse processes a response header through bound plugins
func (pm *PluginManager) ProcessResponse(ctx interface{}, header http.Header, body string) (http.Header, string) {
	for _, h := range pm.ResponseHandlers {
		header, body = h.ProcessResponse(ctx, header, body)
	}
	return header, body
}
