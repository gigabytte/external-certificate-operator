package openssl

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-logr/logr"
)

var (
	execCommand = exec.Command
	readFile    = os.ReadFile
)

// NewOpenSSLRunners creates and initializes a new OpenSSLRunners object
func NewOpenSSLRunners(ctx context.Context, logger logr.Logger) *OpenSSLRunners {
	return &OpenSSLRunners{
		Ctx:    ctx,
		Logger: logger,
		Certificates: &Certificates{
			TLSCrt:    []byte{},
			TLSKey:    []byte{},
			CACrt:     []byte{},
			PemBundle: []byte{},
			CertificateDetails: &CertificateDetails{
				ExpirationTime: 0,
			},
			ConvertedFormats: &ConvertedFormats{
				PFXCertificate: []byte{},
				X509Certificate: &X509Certificate{
					Certificate: []*x509.Certificate{},
					PrivateKey:  nil,
				},
			},
		},
	}
}

// ProcessCertificates runs all functions in the OpenSSLRunners class for each certificate type assuming they aren't an empty byte slice
func (osr *OpenSSLRunners) ProcessCertificates() error {
	// Process TLSCrt
	if len(osr.Certificates.TLSCrt) > 0 {
		osr.Logger.Info("Processing TLSCrt")
		if err := osr.GetCertificateExpiration(osr.Certificates.TLSCrt); err != nil {
			osr.Logger.Error(err, "Failed to get certificate expiration for TLSCrt")
			return err
		}
	} else {
		osr.Logger.Info("TLSCrt is empty, skipping")
	}

	// Process CACrt
	if len(osr.Certificates.CACrt) > 0 {
		osr.Logger.Info("Processing CACrt")
		if err := osr.GetCertificateExpiration(osr.Certificates.CACrt); err != nil {
			osr.Logger.Error(err, "Failed to get certificate expiration for CACrt")
			return err
		}
	} else {
		osr.Logger.Info("CACrt is empty, skipping")
	}

	// Process PemBundle
	if len(osr.Certificates.PemBundle) > 0 {
		osr.Logger.Info("Processing PemBundle")
		if err := osr.DecodePEM(); err != nil {
			osr.Logger.Error(err, "Failed to decode PEM for PemBundle")
			return err
		}
		if err := osr.CreatePKCS12(); err != nil {
			osr.Logger.Error(err, "Failed to create PKCS12 for PemBundle")
			return err
		}
		if len(osr.Certificates.ConvertedFormats.PFXCertificate) > 0 {
			if osr.Certificates.CertificateDetails.ExpirationTime == 0 {
				if err := osr.GetCertificateExpiration(osr.Certificates.ConvertedFormats.PFXCertificate); err != nil {
					osr.Logger.Error(err, "Failed to get certificate expiration for PemBundle")
					return err
				}
			}
		}

	} else {
		osr.Logger.Info("PemBundle is empty, skipping")
	}

	osr.Logger.Info("All certificate types processed successfully")
	return nil
}

// createPKCS12 creates a PKCS#12 file from certificates and private key
func (osr *OpenSSLRunners) CreatePKCS12() error {

	certs := osr.Certificates.ConvertedFormats.X509Certificate.Certificate
	key := osr.Certificates.ConvertedFormats.X509Certificate.PrivateKey

	// Certificate validity check
	for _, cert := range certs {
		if time.Now().Before(cert.NotBefore) || time.Now().After(cert.NotAfter) {
			osr.Logger.Error(nil, "Certificate is either not yet valid or has expired")
			return fmt.Errorf("certificate is either not yet valid or has expired")
		}
	}

	// Create temporary files for key and certificate
	keyFile, err := os.CreateTemp("", "key-*.pem")
	if err != nil {
		osr.Logger.Error(err, "Failed to create temp file for key")
		return fmt.Errorf("failed to create temp file for key: %w", err)
	}
	defer func() {
		if cerr := keyFile.Close(); cerr != nil {
			osr.Logger.Error(cerr, "Failed to close key temp file for cert")
		}
		if cerr := os.Remove(keyFile.Name()); cerr != nil {
			osr.Logger.Error(cerr, "Failed to remove key temp file for cert")
		}
	}()

	certFile, err := os.CreateTemp("", "cert-*.pem")
	if err != nil {
		osr.Logger.Error(err, "Failed to create temp file for cert")
		return fmt.Errorf("failed to create temp file for cert: %w", err)
	}
	defer func() {
		if cerr := certFile.Close(); cerr != nil {
			osr.Logger.Error(cerr, "Failed to close temp file for cert")
		}
		if cerr := os.Remove(certFile.Name()); cerr != nil {
			osr.Logger.Error(cerr, "Failed to remove temp file for cert")
		}
	}()

	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		osr.Logger.Error(nil, "Key is not of type *rsa.PrivateKey")
		return fmt.Errorf("key is not of type *rsa.PrivateKey")
	}

	// Write key to PEM file
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}); err != nil {
		osr.Logger.Error(err, "Failed to write key to PEM")
		return fmt.Errorf("failed to write key to PEM: %w", err)
	}

	// Write certificates to PEM file
	for _, cert := range certs {
		if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
			osr.Logger.Error(err, "Failed to write cert to PEM")
			return fmt.Errorf("failed to write cert to PEM: %w", err)
		}
	}

	// Sync files to ensure data is written
	if err := keyFile.Sync(); err != nil {
		osr.Logger.Error(err, "Failed to sync key file")
		return fmt.Errorf("failed to sync key file: %w", err)
	}
	if err := certFile.Sync(); err != nil {
		osr.Logger.Error(err, "Failed to sync cert file")
		return fmt.Errorf("failed to sync cert file: %w", err)
	}

	// Create output file name
	outputFileName := certFile.Name() + ".pfx"

	// Run OpenSSL command to create PKCS#12 file
	cmd := execCommand("openssl", "pkcs12", "-export", "-out", outputFileName, "-inkey", keyFile.Name(), "-in", certFile.Name(), "-passout", "pass:")
	if err := cmd.Run(); err != nil {
		osr.Logger.Error(err, "OpenSSL command failed")
		return fmt.Errorf("OpenSSL command failed: %w", err)
	}
	// Read the PKCS#12 file
	pfxData, err := readFile(outputFileName)
	if err != nil {
		osr.Logger.Error(err, "Failed to read PFX file")
		return fmt.Errorf("failed to read PFX file: %v", err)
	}
	defer func() {
		if cerr := os.Remove(outputFileName); cerr != nil {
			osr.Logger.Error(cerr, "Failed to remove temp PFX file")
		}
	}()
	osr.Certificates.ConvertedFormats.PFXCertificate = pfxData

	return nil
}

// decodePEM decodes the PEM-encoded certificates and private key
func (osr *OpenSSLRunners) DecodePEM() error {

	var certs []*x509.Certificate
	var key interface{}
	combinedPEM := osr.Certificates.PemBundle

	// Iterate over the PEM blocks
	for {
		block, rest := pem.Decode(combinedPEM)
		if block == nil {
			osr.Logger.Error(nil, "Failed to parse PEM block")
			return fmt.Errorf("failed to parse PEM block")
		}

		switch block.Type {
		case "CERTIFICATE":
			// Parse certificate
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				osr.Logger.Error(err, "Failed to parse certificate")
				return fmt.Errorf("failed to parse certificate: %v", err)
			}
			certs = append(certs, cert)
		case "RSA PRIVATE KEY", "PRIVATE KEY":
			// Parse private key
			var err error
			key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
				if err != nil {
					osr.Logger.Error(err, "Failed to parse private key")
					return fmt.Errorf("failed to parse private key: %v", err)
				}
			}
		}

		combinedPEM = rest
		if len(rest) == 0 {
			break
		}
	}

	// Check if certificates and private key are present
	if len(certs) == 0 || key == nil {
		osr.Logger.Error(nil, "Certificate or key missing in PEM")
		return fmt.Errorf("certificate or key missing in PEM")
	}

	osr.Certificates.ConvertedFormats.X509Certificate = &X509Certificate{
		Certificate: certs,
		PrivateKey:  key,
	}
	osr.Logger.Info("PEM decoded successfully")

	return nil
}

// GetCertificateExpiration returns the expiration date of a certificate as Unix time in seconds (float64)
func (osr *OpenSSLRunners) GetCertificateExpiration(certByte []byte) error {
	osr.Logger.Info("Getting certificate expiration", "certByteLength", len(certByte))

	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "cert-*.crt")
	if err != nil {
		osr.Logger.Error(err, "Failed to create temporary file")
		return err
	}
	defer func() {
		if cerr := os.Remove(tmpfile.Name()); cerr != nil {
			osr.Logger.Error(cerr, "Failed to remove temp file")
		}
	}()

	// Write the certificate string to the temporary file
	if _, err := tmpfile.Write(certByte); err != nil {
		osr.Logger.Error(err, "Failed to write certificate to temporary file")
		return err
	}
	if err := tmpfile.Close(); err != nil {
		osr.Logger.Error(err, "Failed to close temporary file")
		return err
	}
	osr.Logger.Info("Certificate written to temporary file", "fileName", tmpfile.Name())

	// Use OpenSSL to get the expiration date
	cmd := execCommand("openssl", "x509", "-enddate", "-noout", "-in", tmpfile.Name())
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		osr.Logger.Error(err, "Failed to run OpenSSL command")
		return err
	}

	output := out.String()
	osr.Logger.Info("OpenSSL command output", "output", output)

	// Example output: "notAfter=Dec 31 23:59:59 2024 GMT"
	parts := strings.Split(output, "=")
	if len(parts) != 2 {
		osr.Logger.Error(nil, "Unexpected OpenSSL output", "output", output)
		return fmt.Errorf("unexpected output: %s", output)
	}

	dateStr := strings.TrimSpace(parts[1])
	osr.Logger.Info("Parsed date string from OpenSSL output", "dateStr", dateStr)

	expirationDate, err := time.Parse("Jan 2 15:04:05 2006 MST", dateStr)
	if err != nil {
		osr.Logger.Error(err, "Failed to parse expiration date")
		return err
	}

	osr.Certificates.CertificateDetails.ExpirationTime = float64(expirationDate.Unix())
	osr.Logger.Info("Certificate expiration date retrieved successfully", "expirationTime", osr.Certificates.CertificateDetails.ExpirationTime)

	return nil
}
