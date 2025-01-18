package controller

import "time"

const (
	FINALIZER           = "external-certificate.io/finalizer"
	BASE64EN_TEMPLATING = "base64encode"
	PEMTOPFX_TEMPLATING = "pemtopfx"
	baseDelay           = time.Minute * 2
	baseDelayDelete     = time.Second * 30
	maxRetries          = 5
	maxDuration         = 30 * time.Minute

	IMPORT_CERT_SECRET_CRD = "importcertificatesecrets.external-certificate.io"
	EXPORT_CERT_SECRET_CRD = "exportcertificatesecrets.external-certificate.io"
)
