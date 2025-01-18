package openssl

import (
	"context"
	"crypto/x509"
)

type OpenSSLRunners struct {
	Ctx          context.Context
	Certificates *Certificates
}

type Certificates struct {
	TLSCrt             []byte
	TLSKey             []byte
	CACrt              []byte
	PemBundle          []byte
	CertificateDetails *CertificateDetails
	ConvertedFormats   *ConvertedFormats
}

type CertificateDetails struct {
	ExpirationTime float64
}

type ConvertedFormats struct {
	PFXCertificate  []byte
	X509Certificate *X509Certificate
}

type X509Certificate struct {
	Certificate []*x509.Certificate
	PrivateKey  interface{}
}
