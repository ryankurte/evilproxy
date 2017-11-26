/**
 * Logger logs proxied methods for future analysis
 *
 * Copyright 2017 Ryan Kurte
 */

package plugins

import (
	"net/http"
)

type Logger struct {
}

func (l *Logger) ProcessRequest(ctx interface{}, header http.Header, body string) (http.Header, string) {
	return header, body
}

func (l *Logger) ProcessResponse(ctx interface{}, header http.Header, body string) (http.Header, string) {
	return header, body
}
