package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jessevdk/go-flags"

	"github.com/ryankurte/experiments/evilproxy/lib/core"
	"github.com/ryankurte/experiments/evilproxy/lib/ingress"
	"github.com/ryankurte/experiments/evilproxy/lib/plugins"
)

func main() {
	log.Printf("☭ EvilProxy (evpx) ☭")

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
	p.BindPlugin(plugins.NewHSTS())

	// Run the frontend
	go h.Run()

	// Wait for exit signal
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Shutdown the ingress server
	h.Stop()
}
