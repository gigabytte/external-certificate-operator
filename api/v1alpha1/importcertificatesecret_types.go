/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImportAzureKVProvider represents the Azure Key Vault provider configuration
type ImportAzureKVProvider struct {
	VaultUrl             string                 `json:"vaultUrl"`
	ServiceAccountRef    ServiceAccountRef      `json:"serviceAccountRef"`
	CertificateSecretRef []CertificateSecretRef `json:"certificateSecretRef"` // defined in general_types.go
	SecretNamespace      string                 `json:"secretNamespace,omitempty"`
	ScanInterval         int                    `json:"scanInterval,omitempty"`
}

// SecretRef represents a reference to a secret
type SecretsRef struct {
	// Name of secret in Key Vault
	Name string `json:"name"`
	// DestinationKeyName is the key name in the destination secret in k8s
	DestinationKeyName string `json:"destinationKeyName"`
}

// ImportCertificateSecretSpec defines the desired state of ImportCertificateSecret
type ImportCertificateSecretSpec struct {

	// AzureKV is the Azure Key Vault provider configuration
	// def in exportcertificatesecret_types.go
	AzureKV ImportAzureKVProvider `json:"azurekv"`
}

// ImportCertificateSecretStatus defines the observed state of ImportCertificateSecret
type ImportCertificateSecretStatus struct {
	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// RetryCount is the number of retries for processing the secret
	RetryCount int `json:"retryCount,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ImportCertificateSecret is the Schema for the importcertificatesecrets API
type ImportCertificateSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImportCertificateSecretSpec   `json:"spec,omitempty"`
	Status ImportCertificateSecretStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ImportCertificateSecretList contains a list of ImportCertificateSecret
type ImportCertificateSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImportCertificateSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImportCertificateSecret{}, &ImportCertificateSecretList{})
}
