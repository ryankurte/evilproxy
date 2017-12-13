package ingress

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
)

// HTTPFrontend is a http subdomain based re-mapping proxy
type HTTPFrontend struct {
	Proxy
	address, port string
	bindAddress   string
	srv           *http.Server
	bumpTLS       *BumpTLS
}

// NewHTTPFrontend is an http frontend
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

// BindProxy binds the underlying proxy to the frontend
func (h *HTTPFrontend) BindProxy(p Proxy) {
	h.Proxy = p
}

// wrapRequest modifies the underlying request
func (h *HTTPFrontend) wrapRequest(req *http.Request) (*http.Request, error) {
	queryURI, host := req.RequestURI, req.Host

	log.Printf("Query: %s Host: %s", queryURI, host)

	if req.Body == nil {
		return http.NewRequest(req.Method, queryURI, nil)
	}

	return http.NewRequest(req.Method, queryURI, req.Body)
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

type SingleListener struct {
	conn net.Conn
	once sync.Once
}

func NewSingleListener(conn net.Conn) SingleListener {
	sl := SingleListener{
		conn: conn,
		once: sync.Once{},
	}
	return sl
}

func (sl *SingleListener) Accept() (net.Conn, error) {
	var c net.Conn
	sl.once.Do(func() {
		c = sl.conn
	})
	if c != nil {
		return c, nil
	}
	return nil, io.EOF
}

func (sl *SingleListener) Close() error {
	sl.once.Do(func() {
		sl.conn.Close()
	})
	return nil
}

func (sl *SingleListener) Addr() net.Addr {
	return sl.conn.LocalAddr()
}

func (h *HTTPFrontend) handleConnect(w http.ResponseWriter, r *http.Request) {

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "CONNECT webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	config, err := h.bumpTLS.GetConfigByName(r.Host)
	if err != nil {
		http.Error(w, "CONNECT error getting bumpTLS config", http.StatusInternalServerError)
		log.Printf("CONNECT error getting bumpTLS config: %s", err)
		return
	}

	w.WriteHeader(http.StatusOK)

	conn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	listener := NewSingleListener(conn)

	log.Printf("COnfig: %+v", config)

	tlsListener := tls.NewListener(&listener, config)

	go h.srv.Serve(tlsListener)
}

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

func (h *HTTPFrontend) Stop() {
	h.srv.Shutdown(nil)
}
