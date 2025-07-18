package vars

import "time"

const (
	FINALIZER           = "external-certificate.io/finalizer"
	BASE64EN_TEMPLATING = "base64encode"
	PEMTOPFX_TEMPLATING = "pemtopfx"
	BaseDelay           = time.Minute * 2
	BaseDelayDelete     = time.Second * 30
	MaxRetries          = 5
	MaxDuration         = 30 * time.Minute
	MaxStatusCount      = 5

	IMPORT_CERT_SECRET_CRD = "importcertificatesecrets.external-certificate.io"
	EXPORT_CERT_SECRET_CRD = "exportcertificatesecrets.external-certificate.io"
)
