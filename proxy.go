package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/ryankurte/experiments/evilproxy/plugins"
)

type Config struct {
	Address  string `short:"a" long:"address" description:"Address to bind MITM server" default:"localhost"`
	Port     string `short:"p" long:"port" description:"Port on which to bind MITM server" default:"9001"`
	CertFile string `short:"c" long:"tls-cert" description:"TLS certificate file"`
	CertKey  string `short:"k" long:"tls-key" description:"TLS key file"`
}

type Proxy struct {
	plugins.PluginManager
	address, port string
	bindAddress   string
}

func NewProxy(address, port string) *Proxy {
	p := Proxy{
		address:       address,
		port:          port,
		bindAddress:   fmt.Sprintf("%s:%s", address, port),
		PluginManager: plugins.PluginManager{},
	}
	return &p
}

func buildURI(protocol, address, port string) string {
	return fmt.Sprintf("%s-%s-%s", protocol, address, port)
}

func parseURI(uri string) (protocol, address, port string) {
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

func (p *Proxy) wrapRequest(req *http.Request) (*http.Request, error) {
	queryURI, host, from := req.RequestURI, req.Host, req.RemoteAddr

	log.Printf("Request from: %s for host %s with query %s", from, host, queryURI)

	baseURL := strings.Replace(host, "."+p.bindAddress, "", -1)
	protocol, url, port := parseURI(baseURL)

	proxyURL := fmt.Sprintf("%s://%s:%s%s", protocol, url, port, queryURI)

	// Process and update request components
	//processedHeader := p.ProcessRequestHeader(&r.Header)
	//proxyReq.Header = *processedHeader
	if req.Body == nil {
		return http.NewRequest(req.Method, proxyURL, nil)
	}

	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return http.NewRequest(req.Method, proxyURL, nil)
	}

	processedBody := p.ProcessRequestBody(requestBody)
	return http.NewRequest(req.Method, proxyURL, bytes.NewReader(processedBody))
}

func (p *Proxy) handler(wr http.ResponseWriter, req *http.Request) {

	// Create a new (proxied) request object
	proxyReq, err := p.wrapRequest(req)
	if err != nil {
		wr.WriteHeader(http.StatusBadRequest)
		log.Printf("Error creating proxied request: %s", err)
		return
	}

	// Execute proxied request
	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		wr.WriteHeader(http.StatusBadGateway)
		log.Printf("Error calling proxied server: %s", err)
		return
	}

	// Process and update response components
	processedResponseHeader := p.ProcessResponseHeader(&resp.Header)
	resp.Header = *processedResponseHeader
	responseBody, _ := ioutil.ReadAll(resp.Body)
	processedResponseBody := p.ProcessResponseBody(responseBody)

	// Write processed response
	wr.WriteHeader(resp.StatusCode)
	wr.Write(processedResponseBody)

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

	//p.Bind(plugins.NewCORSPlugin("*"))

	p.Run()
}
