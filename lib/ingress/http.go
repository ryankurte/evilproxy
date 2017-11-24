package ingress

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// HTTPFrontend is a http subdomain based re-mapping proxy
type HTTPFrontend struct {
	Proxy
	address, port string
	bindAddress   string
	srv           *http.Server
}

// NewHTTPFrontend is an http frontend
func NewHTTPFrontend(address, port, cert, key string) *HTTPFrontend {
	h := HTTPFrontend{
		address:     address,
		port:        port,
		bindAddress: fmt.Sprintf("%s:%s", address, port),
	}
	return &h
}

func (h *HTTPFrontend) BindProxy(p Proxy) {
	h.Proxy = p
}

// decodeURI decodes a uri string from the http proxy
func (h *HTTPFrontend) decodeURI(uri string) (protocol, address, port string) {
	protocol, address, port = "https", uri, "443"

	s := strings.Split(uri, ".")
	idx := 1

	_, err := strconv.Atoi(s[len(s)-idx])
	if err == nil {
		port = s[len(s)-1]
		address = strings.Replace(address, "."+port, "", -1)
		idx++
	}

	if s[len(s)-idx] == "http" {
		protocol = "http"
		address = strings.Replace(address, ".http", "", -1)
	}

	return protocol, address, port
}

// encodeURI Encode a URI for the http proxy
func (h *HTTPFrontend) encodeURI(protocol, address, port string) string {
	return fmt.Sprintf("%s-%s-%s", protocol, address, port)
}

// wrapRequest modifies the underlying request
func (h *HTTPFrontend) wrapRequest(req *http.Request) (*http.Request, error) {
	queryURI, host := req.RequestURI, req.Host

	log.Printf("Query: %s Host: %s", queryURI, host)

	baseURL := strings.Replace(host, "."+h.bindAddress, "", -1)
	protocol, url, port := h.decodeURI(baseURL)

	proxyURL := fmt.Sprintf("%s://%s:%s%s", protocol, url, port, queryURI)

	log.Printf("Request mapped '%s' to '%s'", queryURI, proxyURL)

	if req.Body == nil {
		return http.NewRequest(req.Method, proxyURL, nil)
	}

	return http.NewRequest(req.Method, proxyURL, req.Body)
}

func (h *HTTPFrontend) wrapResponse(resp *http.Response) (*http.Response, error) {
	return resp, nil
}

func (h *HTTPFrontend) handler(wr http.ResponseWriter, req *http.Request) {

	// Wrap request object for from frontend to backend format
	proxyReq, err := h.wrapRequest(req)
	if err != nil {
		wr.WriteHeader(http.StatusBadGateway)
		log.Printf("Error wrapping proxied request: %s", err)
		return
	}

	// Process request via proxy interface
	proxyResp, err := h.HandleRequest(proxyReq)
	if err != nil {
		wr.WriteHeader(http.StatusBadGateway)
		log.Printf("Error proxying resquest: %s", err)
		return
	}

	// Wrap response object from backend to frontend
	resp, err := h.wrapResponse(proxyResp)
	if err != nil {
		wr.WriteHeader(http.StatusBadGateway)
		log.Printf("Error wrapping proxied response: %s", err)
		return
	}

	// Write processed response
	for k, v := range resp.Header {
		wr.Header().Set(k, v[0])
		for i := 1; i < len(v); i++ {
			wr.Header().Set(k, v[i])
		}
	}
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
	resp.Body.Close()
}

func (h *HTTPFrontend) Run() {
	srv := &http.Server{Addr: h.bindAddress}

	log.Printf("Starting evilproxy at: http://%s", h.bindAddress)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		h.handler(w, r)
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			log.Printf("Httpserver: ListenAndServe() error: %s", err)
		}
	}()

	h.srv = srv
}

func (h *HTTPFrontend) Stop() {
	h.srv.Shutdown(nil)
}
