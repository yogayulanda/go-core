package kafkatls

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/pavlo-v-chernykh/keystore-go/v4"
)

var jksMagic = []byte{0xfe, 0xed, 0xfe, 0xed}

func Load(path, password string) (*tls.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read truststore: %w", err)
	}
	if bytes.HasPrefix(data, jksMagic) {
		return loadJKS(data, password)
	}
	return loadPEM(data)
}

func loadPEM(data []byte) (*tls.Config, error) {
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	var certificatePEM, keyPEM []byte
	for rest := data; len(rest) > 0; {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remaining
		switch block.Type {
		case "CERTIFICATE":
			certificatePEM = append(certificatePEM, pem.EncodeToMemory(block)...)
			certificate, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse PEM trust certificate: %w", err)
			}
			roots.AddCert(certificate)
		case "PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY":
			keyPEM = append(keyPEM, pem.EncodeToMemory(block)...)
		}
	}
	if len(certificatePEM) == 0 {
		return nil, fmt.Errorf("PEM truststore contains no certificates")
	}
	config := &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}
	if len(keyPEM) > 0 {
		certificate, err := tls.X509KeyPair(certificatePEM, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("load PEM client certificate: %w", err)
		}
		config.Certificates = []tls.Certificate{certificate}
	}
	return config, nil
}

func loadJKS(data []byte, password string) (*tls.Config, error) {
	store := keystore.New()
	if err := store.Load(bytes.NewReader(data), []byte(password)); err != nil {
		return nil, fmt.Errorf("decode JKS: %w", err)
	}
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	config := &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}
	for _, alias := range store.Aliases() {
		if store.IsTrustedCertificateEntry(alias) {
			entry, err := store.GetTrustedCertificateEntry(alias)
			if err != nil {
				return nil, fmt.Errorf("read trusted certificate alias %q: %w", alias, err)
			}
			certificate, err := x509.ParseCertificate(entry.Certificate.Content)
			if err != nil {
				return nil, fmt.Errorf("parse trusted certificate alias %q: %w", alias, err)
			}
			config.RootCAs.AddCert(certificate)
		}
		if store.IsPrivateKeyEntry(alias) {
			entry, err := store.GetPrivateKeyEntry(alias, []byte(password))
			if err != nil {
				return nil, fmt.Errorf("read private key alias %q: %w", alias, err)
			}
			privateKey, err := x509.ParsePKCS8PrivateKey(entry.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("parse private key alias %q: %w", alias, err)
			}
			chain := make([][]byte, 0, len(entry.CertificateChain))
			var leaf *x509.Certificate
			for index, item := range entry.CertificateChain {
				parsed, err := x509.ParseCertificate(item.Content)
				if err != nil {
					return nil, fmt.Errorf("parse certificate chain alias %q: %w", alias, err)
				}
				if index == 0 {
					leaf = parsed
				}
				chain = append(chain, item.Content)
			}
			config.Certificates = append(config.Certificates, tls.Certificate{Certificate: chain, PrivateKey: privateKey, Leaf: leaf})
		}
	}
	return config, nil
}
