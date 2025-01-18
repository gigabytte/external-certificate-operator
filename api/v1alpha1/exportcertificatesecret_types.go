package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AzureKVProvider represents the Azure Key Vault provider configuration
type ExportAzureKVProvider struct {
	VaultUrl             string                 `json:"vaultUrl"`
	ServiceAccountRef    ServiceAccountRef      `json:"serviceAccountRef"`
	CertificateSecretRef []CertificateSecretRef `json:"certificateSecretRef"` // defined in general_types.go
	ScanInterval         int                    `json:"scanInterval,omitempty"`
	OnDeletePurge        bool                   `json:"onDeletePurge,omitempty"`
}

// ExportCertificateSecretSpec defines the desired state of ExportCertificateSecret
type ExportCertificateSecretSpec struct {
	AzureKV ExportAzureKVProvider `json:"azurekv"`
}

// ExportCertificateSecretStatus defines the observed state of ExportCertificateSecret
type ExportCertificateSecretStatus struct {
	Conditions         []metav1.Condition     `json:"conditions,omitempty"`
	RetryCount         int                    `json:"retryCount,omitempty"`
	PreviousSecretRefs []CertificateSecretRef `json:"previousSecretRefs,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ExportCertificateSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExportCertificateSecretSpec   `json:"spec,omitempty"`
	Status ExportCertificateSecretStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ExportCertificateSecretList contains a list of ExportCertificateSecret
type ExportCertificateSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExportCertificateSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExportCertificateSecret{}, &ExportCertificateSecretList{})
}
