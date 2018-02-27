package ingress

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
)

// HTTPFrontend is a http proxy based frontend with bump-tls support
type HTTPFrontend struct {
	Proxy
	address, port string
	bindAddress   string
	srv           *http.Server
	bumpTLS       *BumpTLS
}

// NewHTTPFrontend creates a new HTTP frontend
func NewHTTPFrontend(address, port, certFile, keyFile, certDir string) (*HTTPFrontend, error) {
	h := HTTPFrontend{
		address:     address,
		port:        port,
		bindAddress: fmt.Sprintf("%s:%s", address, port),
	}

	b, err := NewBumpTLS(certFile, keyFile, certDir)
	if err != nil {
		return nil, err
	}
	h.bumpTLS = b

	return &h, nil
}

// BindProxy binds the underlying proxy core to the frontend
func (h *HTTPFrontend) BindProxy(p Proxy) {
	h.Proxy = p
}

// wrapRequest modifies the incoming request to meet core proxy requirements
// ie. have a viable query string and body
func (h *HTTPFrontend) wrapRequest(req *http.Request) (*http.Request, error) {
	queryURI, host := req.RequestURI, req.Host

	if !strings.Contains(queryURI, host) {
		queryURI = host + queryURI
	}

	if req.TLS != nil && !strings.HasPrefix(queryURI, "https://") {
		queryURI = "https://" + queryURI
	} else if req.TLS == nil && !strings.HasPrefix(queryURI, "http://") {
		queryURI = "http://" + queryURI
	}

	log.Printf("Request URI: %s", queryURI)

	if req.Body == nil {
		return http.NewRequest(req.Method, queryURI, nil)
	}

	return http.NewRequest(req.Method, queryURI, req.Body)
}

// wrapResponse modifies the outgoing response as is expected by the client
// TODO: probably should wrap request/response to provide contexts and reset queryURIs
func (h *HTTPFrontend) wrapResponse(resp *http.Response) (*http.Response, error) {
	return resp, nil
}

// handler is the incoming request handler
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
		log.Printf("Error proxying request: %s", err)
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

type singleListener struct {
	conn net.Conn
	once sync.Once
}

func newSingleListener(conn net.Conn) singleListener {
	return singleListener{
		conn: conn,
		once: sync.Once{},
	}
}

func (sl *singleListener) Accept() (net.Conn, error) {
	var c net.Conn
	sl.once.Do(func() {
		c = sl.conn
	})
	if c != nil {
		return c, nil
	}
	return nil, io.EOF
}

func (sl *singleListener) Close() error {
	sl.once.Do(func() {
		sl.conn.Close()
	})
	return nil
}

func (sl *singleListener) Addr() net.Addr {
	return sl.conn.LocalAddr()
}

// handleConnect handles the TCP CONNECT method to provide fake TLS termination
func (h *HTTPFrontend) handleConnect(w http.ResponseWriter, r *http.Request) {

	// Check that http response connection is hijackable and fetch hijacker
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "CONNECT webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	// Fetch a TLS configuration
	config, err := h.bumpTLS.GetConfigByName(r.URL.Hostname())
	if err != nil {
		http.Error(w, "CONNECT error getting bumpTLS config", http.StatusInternalServerError)
		log.Printf("CONNECT error getting bumpTLS config: %s", err)
		return
	}

	// Write an http 200 (causes the browser to
	w.WriteHeader(http.StatusOK)

	// Hijack the underlying connection from the http response object
	conn, _, err := hj.Hijack()
	if err != nil {
		log.Printf("CONNECT error hijacking connection: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build a single listener to bind the connection instance to the request
	listener := newSingleListener(conn)

	// Create a TLS listener with the generated TLS configuration
	tlsListener := tls.NewListener(&listener, config)

	// Hand off request to the new TLS listener (in a new process so we can continue accepting requests)
	go h.srv.Serve(tlsListener)
}

// ServeHTTP wraps the underlying proxy handler and provides bump-tls magic
func (h *HTTPFrontend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodConnect:
		h.handleConnect(w, r)
	default:
		h.handler(w, r)
	}
}

// openssl s_client -connect localhost:9001 -debug
// curl --proxy http://localhost:9001 https://google.com -iv

// Run launches the http frontend
func (h *HTTPFrontend) Run() {

	tlsConfig := ConfigTemplate.Clone()
	tlsConfig.GetConfigForClient = h.bumpTLS.GetConfigForClient

	srv := &http.Server{
		Addr:      h.bindAddress,
		Handler:   h,
		TLSConfig: tlsConfig,
	}

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

// Stop shuts down the http frontend
func (h *HTTPFrontend) Stop() {
	h.srv.Shutdown(nil)
}
