/**
 * CORS plugin re-writes CORS preflight headers
 *
 * Copyright 2017 Ryan Kurte
 */

package plugins

import (
	"log"
	"net/http"
)

const (
	CORSHeaderKey = "Access-Control-Allow-Origin"
)

type CORSPlugin struct {
	value string
}

func NewCORSPlugin(value string) *CORSPlugin {
	return &CORSPlugin{
		value: value,
	}
}

func (c *CORSPlugin) HandleResponse(r *http.Response) {
	header := r.Header.Get(CORSHeaderKey)
	log.Printf("CORS Plugin: rewriting header from %s to %s", header, c.value)
	r.Header.Set(CORSHeaderKey, c.value)
}
