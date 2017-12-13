package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jessevdk/go-flags"

	"github.com/ryankurte/experiments/evilproxy/lib/core"
	"github.com/ryankurte/experiments/evilproxy/lib/ingress"
	//"github.com/ryankurte/experiments/evilproxy/lib/plugins"
)

func main() {
	log.Printf("☭ EvilProxy (evpx) ☭")

	// Parse proxy options
	o := core.Options{}
	flags.Parse(&o)

	// Create the core proxy instance
	p := core.NewProxy(o)

	// Bind the http backend into the proxy
	p.BindBackend(&core.HTTPBackend{})

	// Create the frontend
	h, err := ingress.NewHTTPFrontend(o.Address, o.Port, o.CertFile, o.CertKey)
	if err != nil {
		os.Exit(1)
	}

	// Bind the proxy instance to the frontend
	h.BindProxy(p)

	// Bind enabled plugins
	//p.Bind(plugins.NewCORSPlugin("*"))

	// Run the frontend
	go h.Run()

	// Wait for exit signal
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Shutdown the ingress server
	h.Stop()
}
