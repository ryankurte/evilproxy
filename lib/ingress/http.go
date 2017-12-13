package ingress

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"
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
func NewHTTPFrontend(address, port, certFile, keyFile string) (*HTTPFrontend, error) {
	h := HTTPFrontend{
		address:     address,
		port:        port,
		bindAddress: fmt.Sprintf("%s:%s", address, port),
	}

	b, err := NewBumpTLS(certFile, keyFile, "")
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

var conns = make(map[uintptr]net.Conn)
var connMutex = sync.Mutex{}

// writerToConnPrt converts an http.ResponseWriter to a pointer for indexing
func writerToConnPtr(w http.ResponseWriter) uintptr {
	ptrVal := reflect.ValueOf(w)
	val := reflect.Indirect(ptrVal)

	// http.conn
	valconn := val.FieldByName("conn")
	val1 := reflect.Indirect(valconn)

	// net.TCPConn
	ptrRwc := val1.FieldByName("rwc").Elem()
	rwc := reflect.Indirect(ptrRwc)

	// net.Conn
	val1conn := rwc.FieldByName("conn")
	val2 := reflect.Indirect(val1conn)

	return val2.Addr().Pointer()
}

// connToPtr converts a net.Conn into a pointer for indexing
func connToPtr(c net.Conn) uintptr {
	ptrVal := reflect.ValueOf(c)
	return ptrVal.Pointer()
}

// ConnStateListener bound to server and maintains a list of connections by pointer
func (h *HTTPFrontend) ConnStateListener(c net.Conn, cs http.ConnState) {
	connPtr := connToPtr(c)

	// Bind new
	switch cs {
	case http.StateNew:
		log.Printf("CONN Opened: 0x%x\n", connPtr)
		connMutex.Lock()
		conns[connPtr] = c
		connMutex.Unlock()
	case http.StateClosed:
		log.Printf("CONN Closed: 0x%x\n", connPtr)
		connMutex.Lock()
		delete(conns, connPtr)
		connMutex.Unlock()
	}
}

func (h *HTTPFrontend) handleConnect(w http.ResponseWriter, r *http.Request) {

	connPtr := writerToConnPtr(w)

	conn, ok := conns[connPtr]
	if !ok {
		log.Printf("CONNECT error: no matching connection found")
		return
	}

	config, err := h.bumpTLS.GetConfigByName(r.Host)
	if err != nil {
		log.Printf("CONNECT error getting bumpTLS config: %s", err)
		return
	}

	tlsConn := tls.Server(conn, config)

	connMutex.Lock()
	conns[connPtr] = tlsConn
	connMutex.Unlock()

	w.WriteHeader(http.StatusOK)

	tlsConn.Handshake()
}

func (h *HTTPFrontend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodConnect:
		h.handleConnect(w, r)
	default:
		h.handler(w, r)
	}
}

type MyListener struct {
	net.Listener
}

type MyConn struct {
	net.Conn
}

func (h *HTTPFrontend) Run() {
	srv := &http.Server{
		Addr:      h.bindAddress,
		Handler:   h,
		ConnState: h.ConnStateListener,
		TLSConfig: &tls.Config{
			GetConfigForClient: h.bumpTLS.GetConfigForClient,
		}}

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
