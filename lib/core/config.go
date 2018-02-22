package core

// Options for configuring EvilProxy
type Options struct {
	Address string `short:"a" long:"address" description:"Address to bind MITM server" default:"localhost"`
	Port    string `short:"p" long:"port" description:"Port on which to bind MITM server" default:"9001"`
	Mode    string `short:"m" long:"mode" description:"Proxy mode" default:"https" options:"https" options:"socks"`
	CACert  string `short:"c" long:"ca-cert" description:"TLS certificate authority certificate file"`
	CAKey   string `short:"k" long:"ca-key" description:"TLS certificate authority key file"`
	CertDir string `long:"cert-dir" description:"directory for TLS certificate outputs" default:"./certs"`
}
