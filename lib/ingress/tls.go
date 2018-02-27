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
	rnd "math/rand"
	"net/http"
	"os"
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
	SerialNumber: big.NewInt(rnd.Int63()),
	Subject: pkix.Name{
		CommonName:         "EvilProxy (evpx) TLS Interception Proxy",
		Organization:       []string{"EvilCorp"},
		OrganizationalUnit: []string{"Research"},
	},
	NotBefore: time.Now(),
	NotAfter:  time.Now().Add(time.Hour * 24 * 365),

	KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
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

	// Check if default exists
	if certFile == "" && keyFile == "" {
		cert := fmt.Sprintf("%s/%s", outDir, "ca.crt")
		key := fmt.Sprintf("%s/%s", outDir, "ca.key")

		_, err1 := os.Stat(cert)
		_, err2 := os.Stat(key)

		if err1 == nil && err2 == nil {
			certFile = cert
			keyFile = key
		}
	}

	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		os.Mkdir(outDir, 755)
	}

	// Load existing CA if files are specified
	if certFile != "" && keyFile != "" {
		log.Printf("Loading existing CA (reading cert: %s, key: %s)", certFile, keyFile)

		ca, err := b.loadBumpCert(certFile, keyFile)
		if err != nil {
			log.Printf("BumpTLS error loading CA (%s)", err)
			return nil, err
		}

		b.ca = ca

		b.loadCerts(outDir)

	} else {
		certFile, keyFile = fmt.Sprintf("%s/%s", outDir, "ca.crt"), fmt.Sprintf("%s/%s", outDir, "ca.key")
		log.Printf("Generating new CA (writing cert: %s, key: %s)", certFile, keyFile)

		c, err := b.initCA()
		if err != nil {
			return nil, err
		}

		err = ioutil.WriteFile(keyFile, c.keyData, 0644)
		if err != nil {
			return nil, err
		}

		err = ioutil.WriteFile(certFile, c.crtData, 0644)
		if err != nil {
			return nil, err
		}

		b.ca = c
	}

	return &b, nil
}

func (b *BumpTLS) loadBumpCert(certFile, keyFile string) (*BumpCert, error) {
	certPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	var certDERBlock *pem.Block
	certPEMBlock := certPEM
	for {
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock.Type == "CERTIFICATE" {
			break
		}
		if certDERBlock == nil && len(certPEM) == 0 {
			break
		}
	}

	cert, err := x509.ParseCertificate(certDERBlock.Bytes)
	if err != nil {
		return nil, err
	}

	keyPEM, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	keyDERBlock, _ := pem.Decode(keyPEM)
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKCS1PrivateKey(keyDERBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return &BumpCert{
		crt:     cert,
		crtData: certPEM,
		key:     key,
		keyData: keyPEM,
	}, nil
}

func (b *BumpTLS) loadCerts(dir string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".crt") {
			continue
		}

		name := strings.TrimSuffix(f.Name(), ".crt")
		certFile := dir + "/" + name + ".crt"
		keyFile := dir + "/" + name + ".key"

		cert, err := b.loadBumpCert(certFile, keyFile)
		if err != nil {
			log.Printf("Error loading cert %s", err)
			continue
		}

		b.certs[name] = cert
	}

	return nil
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

	certFile, keyFile := fmt.Sprintf("%s/%s.crt", b.outDir, serverName), fmt.Sprintf("%s/%s.key", b.outDir, serverName)

	// Load existing certificate if found
	cert, ok := b.certs[serverName]
	if !ok {
		log.Printf("BumpTLS.GetConfigByName generating certificate for server: %s", serverName)

		cert, err = b.initServer(name)
		if err != nil {
			return nil, err
		}

		err = ioutil.WriteFile(keyFile, cert.keyData, 0644)
		if err != nil {
			return nil, err
		}

		err = ioutil.WriteFile(certFile, cert.crtData, 0644)
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
	template.Issuer = b.ca.crt.Subject

	req, err := http.DefaultClient.Get(fmt.Sprintf("https://%s", name))
	if err != nil {
		log.Printf("BumpTLS.initServer error: %s", err)
		return nil, err
	}

	if req.TLS != nil && len(req.TLS.PeerCertificates) > 1 {
		peer := req.TLS.PeerCertificates[0]

		log.Printf("Peer: %s", peer.Subject.CommonName)
		template.DNSNames = peer.DNSNames
		template.SerialNumber = big.NewInt(rnd.Int63())
		template.Subject = peer.Subject
		template.NotBefore = peer.NotBefore
		template.NotAfter = peer.NotAfter
		template.KeyUsage = peer.KeyUsage
		template.ExtKeyUsage = peer.ExtKeyUsage
		template.BasicConstraintsValid = peer.BasicConstraintsValid
		template.IPAddresses = peer.IPAddresses
	}

	return b.initCert(&template)
}

// initCA creates a CA certificate
func (b *BumpTLS) initCA() (*BumpCert, error) {
	template := certTemplate

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign | x509.KeyUsageCRLSign

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
