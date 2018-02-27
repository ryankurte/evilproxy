package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jessevdk/go-flags"

	"github.com/ryankurte/evilproxy/lib/core"
	"github.com/ryankurte/evilproxy/lib/ingress"
	"github.com/ryankurte/evilproxy/lib/plugins"
)

var version = "undefined"

func main() {
	log.Printf("☭ EvilProxy (version: %s) ☭", version)

	// Parse proxy options
	o := core.Options{}
	_, err := flags.Parse(&o)
	if err != nil {
		os.Exit(0)
	}

	// Create the core proxy instance
	p := core.NewProxy(o)

	// Bind the http backend into the proxy
	p.BindBackend(&core.HTTPBackend{})

	// Create the frontend
	h, err := ingress.NewHTTPFrontend(o.Address, o.Port, o.CACert, o.CAKey, o.CertDir)
	if err != nil {
		log.Printf("Error starting ingress: %s", err)
		os.Exit(1)
	}

	// Bind the proxy instance to the frontend
	h.BindProxy(p)

	// Bind enabled plugins
	if o.BlockAll || o.BlockHSTS {
		p.BindPlugin(plugins.NewHSTS())
	}
	if o.BlockAll || o.BlockCORS {
		p.BindPlugin(plugins.NewCORS("*"))
	}
	if o.BlockAll || o.BlockSRI {
		p.BindPlugin(plugins.NewSRI())
	}

	// Run the frontend
	go h.Run()

	// Wait for exit signal
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Shutdown the ingress server
	h.Stop()
}
