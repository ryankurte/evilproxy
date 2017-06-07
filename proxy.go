package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/jessevdk/go-flags"

	"github.com/ryankurte/experiments/evilproxy/plugins"
	"io/ioutil"
	"strings"
)

type Config struct {
	Address  string `short:"a" long:"address" description:"Address to bind MITM server" default:"localhost"`
	Port     string `short:"p" long:"port" description:"Port on which to bind MITM server" default:"9001"`
	CertFile string `short:"c" long:"tls-cert" description:"TLS certificate file"`
	CertKey  string `short:"k" long:"tls-key" description:"TLS key file"`
}

type Proxy struct {
	address, port    string
	bindAddress      string
	requestHandlers  []plugins.RequestHandler
	responseHandlers []plugins.ResponseHandler
}

func NewProxy(address, port string) *Proxy {
	p := Proxy{
		address:          address,
		port:             port,
		bindAddress:      fmt.Sprintf("%s:%s", address, port),
		requestHandlers:  make([]plugins.RequestHandler, 0),
		responseHandlers: make([]plugins.ResponseHandler, 0),
	}
	return &p
}

func (p *Proxy) BindPlugin(handler interface{}) {
	if reqHandler, ok := handler.(plugins.RequestHandler); ok {
		p.requestHandlers = append(p.requestHandlers, reqHandler)
	}
	if respHandler, ok := handler.(plugins.ResponseHandler); ok {
		p.responseHandlers = append(p.responseHandlers, respHandler)
	}
}

func (p *Proxy) handler(w http.ResponseWriter, r *http.Request) {
	queryURI, host, from := r.RequestURI, r.Host, r.RemoteAddr

	log.Printf("Request from: %s for host %s with query %s", from, host, queryURI)

	baseURL := strings.Replace(host, "."+p.bindAddress, "", -1)
	proxyURL := fmt.Sprintf("https://%s/%s", baseURL, queryURI)

	// Create a new (proxied) request object
	proxyReq, err := http.NewRequest(r.Method, proxyURL, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Error creating proxied request: %s", err)
		return
	}

	for _, h := range p.requestHandlers {
		h.ProcessRequest(proxyReq)
	}

	// Execute proxied request
	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		log.Printf("Error calling proxied server: %s", err)
		return
	}

	// Todo: call response handlers
	for _, h := range p.responseHandlers {
		h.ProcessResponse(resp)
	}

	// Read and return response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error loading response body %s", err)
		return
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body)

}

func (p *Proxy) Run() {
	http.HandleFunc("/", p.handler)

	log.Printf("Starting evilproxy at: http://%s", p.bindAddress)

	err := http.ListenAndServe(p.bindAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.Printf("EvilProxy")

	c := Config{}
	flags.Parse(&c)

	p := NewProxy(c.Address, c.Port)

	p.Run()
}
