package core

type Options struct {
	Address  string `short:"a" long:"address" description:"Address to bind MITM server" default:"localhost"`
	Port     string `short:"p" long:"port" description:"Port on which to bind MITM server" default:"9001"`
	Mode     string `short:"m" long:"mode" description:"Proxy mode" default:"https" options:"https" options:"socks"`
	CertFile string `short:"c" long:"tls-cert" description:"TLS certificate file"`
	CertKey  string `short:"k" long:"tls-key" description:"TLS key file"`
	CertDir  string `long:"cert-dir" description:"directory for TLS certificate outputs" default:"./certs"`
}
