package exportcertctrl

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	certdistributionv1alpha1 "github.com/gigabytte/external-certificate-operator/api/v1alpha1"
)

// SetupWithManager sets up the controller with the Manager.
func (r *ExportCertificateSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Register the CRD type in the scheme
	if err := apiextensionsv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	exportCertPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj, okOld := e.ObjectOld.(*certdistributionv1alpha1.ExportCertificateSecret)
			newObj, okNew := e.ObjectNew.(*certdistributionv1alpha1.ExportCertificateSecret)
			// Trigger reconcile if:
			// 1. Spec changed OR
			// 2. DeletionTimestamp was added (object is being deleted)
			if !okOld || !okNew {
				return false
			}
			if !newObj.DeletionTimestamp.IsZero() {
				return true
			}

			// Otherwise only reconcile on spec changes
			return !reflect.DeepEqual(oldObj.Spec, newObj.Spec)
		},
		CreateFunc:  func(e event.CreateEvent) bool { return true },
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
	}

	secretPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			secret, isSecret := e.Object.(*corev1.Secret)
			if isSecret {
				return r.shouldReconcileSecret(secret)
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			secret, isSecret := e.ObjectNew.(*corev1.Secret)
			if !isSecret {
				return false
			}
			// If the secret is being deleted, we don't need to reconcile
			if !secret.DeletionTimestamp.IsZero() {
				return false
			}

			// Check data changes
			oldSecret := e.ObjectOld.(*corev1.Secret)
			if !reflect.DeepEqual(oldSecret.Data, secret.Data) ||
				!reflect.DeepEqual(oldSecret.Labels, secret.Labels) {
				return r.shouldReconcileSecret(secret)
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&certdistributionv1alpha1.ExportCertificateSecret{},
			builder.WithPredicates(exportCertPredicate)).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				secret, ok := obj.(*corev1.Secret)
				if !ok {
					return nil
				}
				// List all ExportCertificateSecret resources in the same namespace
				var list certdistributionv1alpha1.ExportCertificateSecretList
				if err := r.List(ctx, &list, client.InNamespace(secret.Namespace)); err != nil {
					return nil
				}
				var requests []reconcile.Request
				for _, ecs := range list.Items {
					// Check if this CR references the changed Secret
					for _, ref := range ecs.Spec.AzureKV.CertificateSecretRef {
						if ref.SecretName == secret.Name {
							requests = append(requests, reconcile.Request{
								NamespacedName: client.ObjectKey{
									Name:      ecs.Name,
									Namespace: ecs.Namespace,
								},
							})
							break
						}
					}
				}
				return requests
			}),
			builder.WithPredicates(secretPredicate),
		).
		Complete(r)
}

func (r *ExportCertificateSecretReconciler) shouldReconcileSecret(obj client.Object) bool {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return false
	}

	exportList := &certdistributionv1alpha1.ExportCertificateSecretList{}
	if err := r.List(context.TODO(), exportList, &client.ListOptions{
		Namespace: secret.Namespace,
	}); err != nil {
		return false
	}

	for _, export := range exportList.Items {
		for _, certRef := range export.Spec.AzureKV.CertificateSecretRef {
			if certRef.SecretName == secret.Name {
				return true
			}
		}
	}
	return false
}
