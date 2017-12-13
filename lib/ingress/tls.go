package ingress

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"strings"
	"time"
)

type BumpTLS struct {
	mode   string
	outDir string
	ca     *BumpCert
	certs  map[string]*BumpCert
}

type BumpCert struct {
	crt     *x509.Certificate
	key     *rsa.PrivateKey
	crtData []byte
	keyData []byte
}

var certTemplate = x509.Certificate{
	SerialNumber: big.NewInt(0),
	Subject: pkix.Name{
		Organization: []string{"☭ EvilProxy (evpx) ☭"},
	},
	NotBefore: time.Now(),
	NotAfter:  time.Now().Add(time.Hour * 24),

	KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	BasicConstraintsValid: true,
}

var ConfigTemplate = tls.Config{
	MinVersion:               tls.VersionTLS12,
	CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
	PreferServerCipherSuites: true,
	CipherSuites: []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	},
}

// NewBumpTLS Creates a new BumpTLS instance
func NewBumpTLS(certFile, keyFile, outDir string) (*BumpTLS, error) {
	b := BumpTLS{
		outDir: outDir,
		certs:  make(map[string]*BumpCert),
	}

	if certFile != "" && keyFile != "" {
		tls.LoadX509KeyPair(certFile, keyFile)
		certData, err := ioutil.ReadFile(certFile)
		if err != nil {
			return nil, err
		}

		cert, err := x509.ParseCertificate(certData)
		if err != nil {
			return nil, err
		}

		keyData, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}

		key, err := x509.ParsePKCS1PrivateKey(keyData)
		if err != nil {
			return nil, err
		}

		b.ca = &BumpCert{
			crt:     cert,
			crtData: certData,
			key:     key,
			keyData: keyData,
		}

	} else {
		c, err := b.initCA()
		if err != nil {
			return nil, err
		}

		err = ioutil.WriteFile(fmt.Sprintf("%s/%s", outDir, "ca.key"), c.keyData, 0644)
		if err != nil {
			return nil, err
		}

		err = ioutil.WriteFile(fmt.Sprintf("%s/%s", outDir, "ca.crt"), c.crtData, 0644)
		if err != nil {
			return nil, err
		}

		b.ca = c
	}

	return &b, nil
}

// GetConfigForClient generates a configuration for the server the client is attempting to connect to
func (b *BumpTLS) GetConfigForClient(info *tls.ClientHelloInfo) (*tls.Config, error) {
	return b.GetConfigByName(info.ServerName)
}

// GetConfigByName generates a configuration for the server the client is attempting to connect to
func (b *BumpTLS) GetConfigByName(name string) (*tls.Config, error) {
	cfg := ConfigTemplate.Clone()
	var err error

	serverName := strings.ToLower(name)
	log.Printf("BumpTLS.GetConfigByName for server: %s", serverName)

	// Load existing certificate if found
	cert, ok := b.certs[serverName]
	if !ok {
		cert, err = b.initServer(name)
		if err != nil {
			return nil, err
		}

		b.certs[serverName] = cert
	}

	tlsCert, err := tls.X509KeyPair(cert.crtData, cert.keyData)
	if err != nil {
		log.Printf("BumpTLS.GetConfigByName error: %s", err)
		return nil, err
	}

	cfg.Certificates = []tls.Certificate{tlsCert}

	return cfg, nil
}

// initServer creates a certificate for the requested
func (b *BumpTLS) initServer(name string) (*BumpCert, error) {
	template := certTemplate
	template.DNSNames = []string{name}

	return b.initCert(&template)
}

// initCA creates a CA certificate
func (b *BumpTLS) initCA() (*BumpCert, error) {
	template := certTemplate

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	return b.initCert(&template)
}

// initCert creates a certificate from the provided template
func (b *BumpTLS) initCert(template *x509.Certificate) (*BumpCert, error) {

	if template == nil {
		return nil, fmt.Errorf("Certificate template required")
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Printf("BumpTLS init error: %s", err)
		return nil, err
	}
	keyDer := x509.MarshalPKCS1PrivateKey(key)

	var crtDer []byte
	if b.ca == nil {
		crtDer, err = x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	} else {
		crtDer, err = x509.CreateCertificate(rand.Reader, template, b.ca.crt, key.Public(), b.ca.key)
	}
	if err != nil {
		log.Printf("BumpTLS error creating certificate: %s", err)
		return nil, err
	}

	certPem := bytes.NewBuffer(nil)
	pem.Encode(certPem, &pem.Block{Type: "CERTIFICATE", Bytes: crtDer})

	keyPem := bytes.NewBuffer(nil)
	pem.Encode(keyPem, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyDer})

	crt, err := x509.ParseCertificate(crtDer)
	if err != nil {
		log.Printf("BumpTLS error parsing created certificate: %s", err)
		return nil, err
	}

	if b.ca != nil {
		//crtData = append(crtData, b.ca.crtData...)
		certPem.Write(b.ca.crtData)
	}

	return &BumpCert{crt: crt, key: key, crtData: certPem.Bytes(), keyData: keyPem.Bytes()}, nil
}
