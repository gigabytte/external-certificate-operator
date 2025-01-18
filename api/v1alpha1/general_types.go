package v1alpha1

// ServiceAccountRef represents a reference to a Kubernetes service account
type ServiceAccountRef struct {
	Name      string   `json:"name"`
	Audiences []string `json:"audiences,omitempty"`
}

// CertificateSecretRef represents the reference to the certificate secret
type CertificateSecretRef struct {
	KVSecretName string `json:"kvSecretName,omitempty"`
	Type         string `json:"type,omitempty"`
	SecretKey    string `json:"secretKey"`
	SecretName   string `json:"secretName"`
	Templating   string `json:"templating,omitempty"` // usage only available for export
}
