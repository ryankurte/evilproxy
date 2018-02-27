/**
 * SRI plugin re-writes or recomputes SubResource Integrity Tags for injected files
 *
 * Copyright 2017 Ryan Kurte
 */

package plugins

import (
	"net/http"
	"regexp"
)

const (
	sha256 = "sha256"
	sha384 = "sha384"
	sha512 = "sha512"
)

var sriExp = regexp.MustCompile(`(integrity\=\"[a-z0-9A-Z\-\+\/]+"[\n\r\s]*)`)

// SRI plugin strips SRI tags from http response bodies
type SRI struct {
}

// NewSRI creates a new instance of the SRI strip plugin
func NewSRI() *SRI {
	return &SRI{}
}

// ProcessResponse removes HSTS headers from a proxied response
func (s *SRI) ProcessResponse(ctx interface{}, header http.Header, body string) (http.Header, string) {
	body = string(sriExp.ReplaceAll([]byte(body), []byte{}))
	return header, body
}
