/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func __createCert(t *testing.T, dir string) (certPath, keyPath string) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("serial: %v", err)
	}

	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "last-updated",
		},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}

	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	certOut, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("create cert file: %v", err)
	}
	defer func() { _ = certOut.Close() }()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("write cert pem: %v", err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	defer func() { _ = keyOut.Close() }()

	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		t.Fatalf("write key pem: %v", err)
	}

	return certPath, keyPath
}

func newReturnsErrorWhenConfigIsEmpty(t *testing.T) {
	srv, err := New(context.TODO(), &Config{}, &TLS{}, &Profiler{})
	assert.Nil(t, srv, "service should not be available")
	assert.Error(t, err, "method should fail")
	assert.ErrorContains(t, err, "config must not be empty", "error message should match")
}

func newReturnsErrorWhenTlsIsEmpty(t *testing.T) {
	srv, err := New(context.TODO(), &Config{UpdateOnCreate: true}, &TLS{}, &Profiler{})
	assert.Nil(t, srv, "service should not be available")
	assert.Error(t, err, "method should fail")
	assert.ErrorContains(t, err, "tls must not be empty", "error message should match")
}

func newReturnsErrorWhenConfigIsInvalid(t *testing.T) {
	assets := t.TempDir()
	cert, key := __createCert(t, assets)

	srv, err := New(
		context.TODO(),
		&Config{UpdateOnCreate: true},
		&TLS{
			Key:         key,
			Certificate: cert,
		},
		&Profiler{},
	)
	assert.Nil(t, srv, "service should not be available")
	assert.Error(t, err, "method should fail")
	assert.ErrorContains(t, err, "annotationKey is required", "error message should match")
}

func newReturnsServiceWhenConfigIsValid(t *testing.T) {
	assets := t.TempDir()
	cert, key := __createCert(t, assets)

	srv, err := New(
		context.TODO(),
		&Config{
			AnnotationKey: "updateTimestamp",
			Resources:     []Resource{{Group: "apps", Resource: "deployments"}},
		},
		&TLS{
			Key:         key,
			Certificate: cert,
		},
		&Profiler{},
	)
	assert.NotNil(t, srv, "service should be available")
	assert.NoError(t, err, "method should fail")
	assert.IsType(t, &Service{}, srv, "should return service")
}

func runReturnsErrorWhenTlsIsInvalid(t *testing.T) {
	srv, err := New(
		context.TODO(),
		&Config{
			AnnotationKey: "updateTimestamp",
			Resources:     []Resource{{Group: "apps", Resource: "deployments"}},
		},
		&TLS{
			Key:         "key.pem",
			Certificate: "cert.pem",
		},
		&Profiler{},
	)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	err = srv.Run()
	assert.Error(t, err, "method should fail")
	assert.ErrorContains(t, err, "open cert.pem: no such file or directory", "error message should match")
}

func runListensAndServesWhenTlsIsValid(t *testing.T) {
	assets := t.TempDir()
	cert, key := __createCert(t, assets)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	srv, err := New(
		ctx,
		&Config{
			AnnotationKey: "updateTimestamp",
			Resources:     []Resource{{Group: "apps", Resource: "deployments"}},
		},
		&TLS{
			Key:         key,
			Certificate: cert,
		},
		&Profiler{
			Enable: true,
		},
	)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	err = srv.Run()
	assert.NoError(t, err, "method should not fail")
}

func TestService(t *testing.T) {
	t.Run("service.New returns error when config is empty", newReturnsErrorWhenConfigIsEmpty)
	t.Run("service.New returns error when TLS is empty", newReturnsErrorWhenTlsIsEmpty)
	t.Run("service.New returns error when config is invalid", newReturnsErrorWhenConfigIsInvalid)
	t.Run("service.New returns service when config is valid", newReturnsServiceWhenConfigIsValid)
	t.Run("service.Run returns error when TLS is invalid", runReturnsErrorWhenTlsIsInvalid)
	t.Run("service.Run listens and serves when TLS is valid", runListensAndServesWhenTlsIsValid)
}
