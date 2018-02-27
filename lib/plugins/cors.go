/**
 * CORS plugin re-writes CORS preflight headers
 *
 * Copyright 2017 Ryan Kurte
 */

package plugins

import (
	"net/http"
)

const (
	corsHeaderKey = "Access-Control-Allow-Origin"
)

// CORS plugin strips (or replaces) Cross Origin Resource Sharing (CORS) headers
type CORS struct {
	base
	value string
}

// NewCORS creates a new instance of the CORS plugin
func NewCORS(value string) *CORS {
	return &CORS{
		base:  newBase("cors"),
		value: value,
	}
}

// ProcessResponse strips CORS headers from proxied responses
func (c *CORS) ProcessResponse(ctx interface{}, header http.Header, body string) (http.Header, string) {
	v := header.Get(corsHeaderKey)
	if v != "" {
		c.WithField(corsHeaderKey, v).Printf("rewriting header")
		if c.value != "" {
			header.Set(corsHeaderKey, c.value)
		} else {
			header.Del(corsHeaderKey)
		}
	}

	return header, body
}
