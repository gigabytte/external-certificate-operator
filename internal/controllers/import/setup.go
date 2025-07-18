package importcertctrl

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	certdistributionv1alpha1 "github.com/gigabytte/external-certificate-operator/api/v1alpha1"
)

func (r *ImportCertificateSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Register the CRD type in the scheme
	if err := apiextensionsv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	// Predicate to filter Secret objects based on ImportCertificateSecret references
	namePredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.shouldReconcile(e.Object)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&certdistributionv1alpha1.ImportCertificateSecret{}).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})). // Ignore status updates to prevent unnecessary reconciliations
		Watches(
			&corev1.Secret{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(namePredicate),
		).
		Complete(r)
}

func (r *ImportCertificateSecretReconciler) shouldReconcile(object client.Object) bool {
	if secret, ok := object.(*corev1.Secret); ok {
		importCertSecretList := &certdistributionv1alpha1.ImportCertificateSecretList{}
		if err := r.List(context.TODO(), importCertSecretList); err != nil {
			return false
		}

		for _, importCertSecret := range importCertSecretList.Items {
			secretNamespace := importCertSecret.Namespace
			if importCertSecret.Spec.AzureKV.SecretNamespace != "" {
				secretNamespace = importCertSecret.Spec.AzureKV.SecretNamespace
			}
			for _, certRef := range importCertSecret.Spec.AzureKV.CertificateSecretRef {
				if secret.Name == certRef.SecretName && secret.Namespace == secretNamespace {
					return true
				}
			}
		}
	}
	return false
}
