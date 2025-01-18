package v1alpha1

import (
	"regexp"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *ExportCertificateSecret) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

var _ webhook.Defaulter = &ExportCertificateSecret{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ExportCertificateSecret) Default() {
	if r.Spec.AzureKV.ScanInterval == 0 {
		r.Spec.AzureKV.ScanInterval = 30
	}
	// Set OnDeletePurge to true by default
	if !r.Spec.AzureKV.OnDeletePurge {
		r.Spec.AzureKV.OnDeletePurge = true
	}
	for i := range r.Spec.AzureKV.CertificateSecretRef {
		certRef := &r.Spec.AzureKV.CertificateSecretRef[i]
		// Build the KVSecretName if not provided
		// Sanitize the KVSecretName to be a valid KeyVault secret name
		if certRef.KVSecretName == "" {
			certRef.KVSecretName = r.sanitizeSecretNameString(certRef.SecretName + "-" + certRef.Type)
		} else {
			certRef.KVSecretName = r.sanitizeSecretNameString(certRef.KVSecretName)
		}
		if certRef.SecretKey == TLSCRT || certRef.SecretKey == TLSKEY || certRef.SecretKey == CACRT {
			certRef.Type = "crt"
		}
		if certRef.SecretKey == COMBINEDPEM {
			certRef.Type = "pem"
			if certRef.Templating == "" {
				certRef.Templating = PEMTOPFX_TEMPLATING
			}
		}
	}
}

var _ webhook.Validator = &ExportCertificateSecret{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ExportCertificateSecret) ValidateCreate() (admission.Warnings, error) {
	if err := r.validateCertificateSecretRef(); err != nil {
		return nil, err
	}
	if err := r.validateSecretNames(); err != nil {
		return nil, err
	}
	if err := r.validateUniqueKVSecretNames(); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ExportCertificateSecret) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	if err := r.validateCertificateSecretRef(); err != nil {
		return nil, err
	}
	if err := r.validateSecretNames(); err != nil {
		return nil, err
	}
	if err := r.validateUniqueKVSecretNames(); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ExportCertificateSecret) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

// validateCertificateSecretRef validates that only one of crt or pem is defined and checks templating values
func (r *ExportCertificateSecret) validateCertificateSecretRef() error {
	var allErrs field.ErrorList

	allowedKeys := map[string]bool{
		TLSKEY:      true,
		TLSCRT:      true,
		CACRT:       true,
		COMBINEDPEM: true,
	}

	for i, certRef := range r.Spec.AzureKV.CertificateSecretRef {
		if !allowedKeys[certRef.SecretKey] {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("azureKV").Child("certificateSecretRef").Index(i).Child("secretKey"),
				certRef.SecretKey,
				"Invalid secret key. Only tls.key, tls.crt, ca.crt, and tls-combined.pem are allowed",
			))
		}
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(schema.GroupKind{Group: "external-certificate.io", Kind: "ExportCertificateSecret"}, r.Name, allErrs)
}

// validateSecretNames validates that all SecretName values in CertificateSecretRef are the same
func (r *ExportCertificateSecret) validateSecretNames() error {
	if len(r.Spec.AzureKV.CertificateSecretRef) == 0 {
		return nil
	}

	secretNameMap := make(map[string]bool)
	for _, certRef := range r.Spec.AzureKV.CertificateSecretRef {
		secretNameMap[certRef.SecretName] = true
	}

	if len(secretNameMap) > 1 {
		var allErrs field.ErrorList
		for i, certRef := range r.Spec.AzureKV.CertificateSecretRef {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("azureKV").Child("certificateSecretRef").Index(i).Child("secretName"),
				certRef.SecretName,
				"All SecretName values in CertificateSecretRef must be the same",
			))
		}
		return apierrors.NewInvalid(schema.GroupKind{Group: "external-certificate.io", Kind: "ExportCertificateSecret"}, r.Name, allErrs)
	}

	return nil
}

// validateUniqueKVSecretNames validates that all KVSecretName values in CertificateSecretRef are unique
func (r *ExportCertificateSecret) validateUniqueKVSecretNames() error {
	if len(r.Spec.AzureKV.CertificateSecretRef) == 0 {
		return nil
	}

	kvSecretNameMap := make(map[string]bool)
	var allErrs field.ErrorList

	for i, certRef := range r.Spec.AzureKV.CertificateSecretRef {
		if _, exists := kvSecretNameMap[certRef.KVSecretName]; exists {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("azureKV").Child("certificateSecretRef").Index(i).Child("kvSecretName"),
				certRef.KVSecretName,
				"KVSecretName values must be unique",
			))
		} else {
			kvSecretNameMap[certRef.KVSecretName] = true
		}
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(schema.GroupKind{Group: "external-certificate.io", Kind: "ExportCertificateSecret"}, r.Name, allErrs)
}

// sanitizeSecretNameString corrects the secret name to be a valid KeyVault secret name
func (r *ExportCertificateSecret) sanitizeSecretNameString(input string) string {
	re := regexp.MustCompile("[^0-9a-zA-Z-]+")
	return re.ReplaceAllString(input, "")
}
