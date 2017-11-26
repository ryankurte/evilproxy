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

func (c *CORSPlugin) ProcessResponse(ctx interface{}, header http.Header, body string) (http.Header, string) {
	value := header.Get(CORSHeaderKey)
	log.Printf("CORS Plugin: rewriting header from %s to %s", value, c.value)
	header.Set(CORSHeaderKey, c.value)
	return header, body
}
