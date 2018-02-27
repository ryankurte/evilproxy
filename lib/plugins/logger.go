/**
 * Logger logs proxied methods for future analysis
 *
 * Copyright 2017 Ryan Kurte
 */

package plugins

import (
	"net/http"
)

// Logger plugin logs requests and responses
type Logger struct {
	base
}

// NewLogger creates a new logger instance
// TODO: verbosity options and save to file
func NewLogger() Logger {
	return Logger{base: newBase("logger")}
}

// ProcessRequest logs a proxy request
func (l *Logger) ProcessRequest(ctx interface{}, header http.Header, body string) (http.Header, string) {
	// TODO: log request
	return header, body
}

// ProcessResponse logs a proxy response
func (l *Logger) ProcessResponse(ctx interface{}, header http.Header, body string) (http.Header, string) {
	// TODO: log responses
	return header, body
}
