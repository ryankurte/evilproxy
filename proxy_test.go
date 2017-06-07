package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxy(t *testing.T) {
	p := NewProxy("localhost", "9002")

	t.Run("Passes basic GET requests", func(t *testing.T) {
		req, err := http.NewRequest("GET", "www.google.com.http.80.localhost:9002", nil)
		req.RemoteAddr = "localhost:9027"
		assert.Nil(t, err)

		wrapped, err := p.wrapRequest(req)
		assert.Nil(t, err)

		assert.EqualValues(t, "http://google.com:80", wrapped.URL)

	})

}
