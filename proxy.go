package main

import (
	"fmt"
	"net/http"

	"github.com/jessevdk/go-flags"
	"log"
)

type Config struct {
	Address  string `short:"a" long:"address" description:"Address to bind MITM server" default:"localhost"`
	Port     string `short:"p" long:"port" description:"Port on which to bind MITM server" default:"9001"`
	CertFile string `short:"c" long:"tls-cert" description:"TLS certificate file"`
	CertKey  string `short:"k" long:"tls-key" description:"TLS key file"`
}

type RequestHandler interface {
	ProcessRequest(req *http.Request) http.Request
}

type ResponseHandler interface {
	ProcessResponse(req *http.Request) http.Request
}

func baseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func main() {
	log.Printf("EvilProxy")

	c := Config{}
	flags.Parse(&c)

	http.HandleFunc("/", baseHandler)

	bindAddress := fmt.Sprintf("%s:%s", c.Address, c.Port)
	log.Printf("Starting evilproxy at: http://%s", bindAddress)

	err := http.ListenAndServe(bindAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}
