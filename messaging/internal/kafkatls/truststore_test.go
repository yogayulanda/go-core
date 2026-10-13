package kafkatls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pavlo-v-chernykh/keystore-go/v4"
)

func testCertificate(t *testing.T) ([]byte, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	certificate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "kafka-test"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, certificate, certificate, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return der, key
}

func TestLoadJKS(t *testing.T) {
	der, _ := testCertificate(t)
	store := keystore.New()
	if err := store.SetTrustedCertificateEntry("kafka-root", keystore.TrustedCertificateEntry{Certificate: keystore.Certificate{Type: "X.509", Content: der}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "kafka.jks")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Store(file, []byte("changeit")); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	config, err := Load(path, "changeit")
	if err != nil {
		t.Fatal(err)
	}
	if config.RootCAs == nil || len(config.Certificates) != 0 {
		t.Fatalf("unexpected JKS trust-only config: %#v", config)
	}
}

func TestLoadPEM(t *testing.T) {
	der, key := testCertificate(t)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	data := append(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})...)
	path := filepath.Join(t.TempDir(), "kafka.pem")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := Load(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if config.RootCAs == nil || len(config.Certificates) != 1 {
		t.Fatalf("PEM trust/client certificate was not loaded: %#v", config)
	}
}

func TestLoadRejectsInvalidPEM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.pem")
	if err := os.WriteFile(path, []byte("not-a-certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, ""); err == nil {
		t.Fatal("expected invalid PEM truststore to fail")
	}
}
