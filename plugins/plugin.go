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

// RequestHandler interface implemented by plugins that re-write request information
type RequestHandler interface {
	ProcessRequest(req *http.Request) *http.Request
}

// ResponseHandler interface implemented by plugins that re-write response information
type ResponseHandler interface {
	ProcessResponse(req *http.Response) *http.Response
}
