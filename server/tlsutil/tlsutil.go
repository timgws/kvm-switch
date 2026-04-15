package tlsutil

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/timgws/kvm-switch/server/config"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func Ensure(args config.TlsConfig) (certPath, keyPath, caPath string, err error) {
	if args.Dir == "" {
		args.Dir = "fence-secrets"
	}
	if args.CertFile == "" {
		args.CertFile = "server.pem"
	}
	if args.KeyFile == "" {
		args.KeyFile = "server-key.pem"
	}
	if args.CaFile == "" {
		args.CaFile = "ca.pem"
	}

	if err := os.MkdirAll(args.Dir, 0o700); err != nil {
		return "", "", "", err
	}

	certPath = filepath.Join(args.Dir, args.CertFile)
	keyPath = filepath.Join(args.Dir, args.KeyFile)
	caPath = filepath.Join(args.Dir, args.CaFile)

	// If files exist, assume ok.
	if fileExists(certPath) && fileExists(keyPath) && fileExists(caPath) {
		return certPath, keyPath, caPath, nil
	}

	caPub, caPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", "", err
	}

	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Fence Local CA"},
		NotBefore:             time.Now().Add(-5 * time.Minute),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, caPub, caPriv)
	if err != nil {
		return "", "", "", err
	}

	srvPub, srvPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", "", err
	}

	srvTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "Fence Server"},
		NotBefore:    time.Now().Add(-5 * time.Minute),
		NotAfter:     time.Now().AddDate(5, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     args.DNSNames,
		IPAddresses:  args.RealIPAddresses,
	}

	srvDER, err := x509.CreateCertificate(rand.Reader, srvTmpl, caTmpl, srvPub, caPriv)
	if err != nil {
		return "", "", "", err
	}

	if err := writePEM(caPath, 0o644, "CERTIFICATE", caDER); err != nil {
		return "", "", "", err
	}

	if err := writePEM(certPath, 0o644, "CERTIFICATE", srvDER); err != nil {
		return "", "", "", err
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(srvPriv)
	if err != nil {
		return "", "", "", err
	}

	if err := writePEM(keyPath, 0o600, "PRIVATE KEY", keyBytes); err != nil {
		return "", "", "", err
	}

	return certPath, keyPath, caPath, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func writePEM(path string, mode os.FileMode, typ string, der []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: typ, Bytes: der})
}

func ServerTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
}
