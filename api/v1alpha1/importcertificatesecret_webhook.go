package v1alpha1

import (
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var allowedNamespaces []string

func SetAllowedNamespaces(namespaces []string) {
	allowedNamespaces = namespaces
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *ImportCertificateSecret) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

var _ webhook.Defaulter = &ImportCertificateSecret{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ImportCertificateSecret) Default() {
	if r.Spec.AzureKV.ScanInterval == 0 {
		r.Spec.AzureKV.ScanInterval = 30
	}
}

var _ webhook.Validator = &ImportCertificateSecret{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ImportCertificateSecret) ValidateCreate() (admission.Warnings, error) {
	return nil, r.validateSecretNameSpace()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ImportCertificateSecret) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	return nil, r.validateSecretNameSpace()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ImportCertificateSecret) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

// validateSecretNameSpace validates that only one of crt or pem is defined and checks templating values
func (r *ImportCertificateSecret) validateSecretNameSpace() error {
	if r.Spec.AzureKV.SecretNamespace == "" {
		return nil
	}
	if strings.Join(allowedNamespaces, ", ") == "" {
		return nil
	}
	for _, ns := range allowedNamespaces {
		if r.Spec.AzureKV.SecretNamespace == ns {
			return nil
		}
	}
	return apierrors.NewInvalid(
		schema.GroupKind{Group: "external-certificate.io", Kind: "ImportCertificateSecret"},
		r.Name,
		field.ErrorList{
			field.Invalid(field.NewPath("spec").Child("azureKV").Child("SecretNamespace"), r.Spec.AzureKV.SecretNamespace, "Values for Spec.AzureKV.SecretNamespace allowed: "+strings.Join(allowedNamespaces, ", ")),
		},
	)
}
