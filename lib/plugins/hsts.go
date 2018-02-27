package plugins

import (
	"net/http"
)

// HSTS is an HTTP Strict Transport Security stripping plugin
type HSTS struct {
	base
}

const hstsKey = "Strict-Transport-Security"

// NewHSTS creates a new plugin instance
func NewHSTS() *HSTS {
	return &HSTS{newBase("hsts")}
}

// ProcessResponse removes HSTS headers from a proxied response
func (s *HSTS) ProcessResponse(ctx interface{}, header http.Header, body string) (http.Header, string) {
	v := header.Get(hstsKey)
	if v != "" {
		s.WithField("hstsKey", v).Printf("stripped")
		header.Del(hstsKey)
	}

	return header, body
}
