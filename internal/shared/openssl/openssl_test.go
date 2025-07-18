package openssl

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gigabytte/external-certificate-operator/internal/shared/log"
)

func TestNewOpenSSLRunners(t *testing.T) {
	ctx := context.TODO()
	osr := NewOpenSSLRunners(ctx, log.FromContext(ctx))
	assert.NotNil(t, osr)
	assert.NotNil(t, osr.Certificates)
	assert.NotNil(t, osr.Certificates.CertificateDetails)
	assert.NotNil(t, osr.Certificates.ConvertedFormats)
	assert.NotNil(t, osr.Certificates.ConvertedFormats.X509Certificate)
}

func TestProcessCertificates(t *testing.T) {
	ctx := context.TODO()
	osr := NewOpenSSLRunners(ctx, log.FromContext(ctx))

	// Generate test certificates
	tlsCrt, tlsKey, caCrt, pemBundle := generateTestCertificates(t)

	// Mock exec.Command for GetCertificateExpiration and CreatePKCS12
	execCommand = func(name string, arg ...string) *exec.Cmd {
		if name == "openssl" && arg[0] == "x509" {
			return exec.Command("echo", "notAfter=Dec 31 23:59:59 2024 GMT")
		} else if name == "openssl" && arg[0] == "pkcs12" {
			tmpFile, err := os.CreateTemp("", "cert-*.pem.pfx")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer func() {
				if cerr := tmpFile.Close(); cerr != nil {
					t.Fatalf("Failed to close temp file: %v", cerr)
				}
			}()
			// Write some dummy data to the file to simulate the PFX file creation
			if _, err := tmpFile.Write([]byte("dummy pfx data")); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			return exec.Command("echo", tmpFile.Name())
		}
		return exec.Command(name, arg...)
	}

	// Mock readFile function
	readFile = func(filename string) ([]byte, error) {
		return []byte("dummy pfx data"), nil
	}

	// Test with empty certificates
	err := osr.ProcessCertificates()
	assert.NoError(t, err)

	// Test with valid certificates
	osr.Certificates.TLSCrt = tlsCrt
	err = osr.ProcessCertificates()
	assert.NoError(t, err)

	osr.Certificates.TLSKey = tlsKey
	err = osr.ProcessCertificates()
	assert.NoError(t, err)

	osr.Certificates.CACrt = caCrt
	err = osr.ProcessCertificates()
	assert.NoError(t, err)

	osr.Certificates.PemBundle = pemBundle
	err = osr.ProcessCertificates()
	assert.NoError(t, err)
}

func TestCreatePKCS12(t *testing.T) {
	ctx := context.TODO()
	osr := NewOpenSSLRunners(ctx, log.FromContext(ctx))

	// Mock certificates and generate a valid private key
	cert := &x509.Certificate{NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	osr.Certificates.ConvertedFormats.X509Certificate.Certificate = []*x509.Certificate{cert}
	osr.Certificates.ConvertedFormats.X509Certificate.PrivateKey = key

	// Mock exec.Command
	execCommand = func(name string, arg ...string) *exec.Cmd {
		tmpFile, err := os.CreateTemp("", "cert-*.pem.pfx")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer func() {
			if cerr := tmpFile.Close(); cerr != nil {
				t.Fatalf("Failed to close temp file: %v", cerr)
			}
		}()
		// Write some dummy data to the file to simulate the PFX file creation
		if _, err := tmpFile.Write([]byte("dummy pfx data")); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		return exec.Command("echo", tmpFile.Name())
	}

	// Mock readFile function
	readFile = func(filename string) ([]byte, error) {
		return []byte("dummy pfx data"), nil
	}

	err = osr.CreatePKCS12()
	assert.NoError(t, err)
}

func TestDecodePEM(t *testing.T) {
	ctx := context.TODO()
	osr := NewOpenSSLRunners(ctx, log.FromContext(ctx))

	// Generate test certificates and use PEM bundle
	_, _, _, pemBundle := generateTestCertificates(t)
	osr.Certificates.PemBundle = pemBundle
	err := osr.DecodePEM()
	assert.NoError(t, err)
	assert.NotNil(t, osr.Certificates.ConvertedFormats.X509Certificate)
	assert.NotNil(t, osr.Certificates.ConvertedFormats.X509Certificate.Certificate)
	assert.NotNil(t, osr.Certificates.ConvertedFormats.X509Certificate.PrivateKey)
}

func TestGetCertificateExpiration(t *testing.T) {
	ctx := context.TODO()
	osr := NewOpenSSLRunners(ctx, log.FromContext(ctx))

	// Generate test certificates
	tlsCrt, _, _, _ := generateTestCertificates(t)

	// Mock exec.Command
	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("echo", "notAfter=Dec 31 23:59:59 2024 GMT")
	}

	err := osr.GetCertificateExpiration(tlsCrt)
	assert.NoError(t, err)
	assert.Equal(t, float64(1735689599), osr.Certificates.CertificateDetails.ExpirationTime)
}
