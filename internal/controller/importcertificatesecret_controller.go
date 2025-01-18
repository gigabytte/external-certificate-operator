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

package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	certdistributionv1alpha1 "github.com/gigabytte/external-certificate-operator/api/v1alpha1"
	"github.com/gigabytte/external-certificate-operator/internal/log"
	akv "github.com/gigabytte/external-certificate-operator/internal/providers/azure/akv"
	azureauth "github.com/gigabytte/external-certificate-operator/internal/providers/azure/auth"
)

// ImportCertificateSecretReconciler reconciles a ImportCertificateSecret object
type ImportCertificateSecretReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	Ctx                     context.Context
	AkvClient               *akv.KeyVault
	KubeClient              kubernetes.Interface
	ImportCertificateSecret *certdistributionv1alpha1.ImportCertificateSecret
}

const (
	SECRET_CHG_ANNOTATION = "external-certificate.io/secret-state-chg"
)

//+kubebuilder:rbac:groups=external-certificate.io,resources=importcertificatesecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=external-certificate.io,resources=importcertificatesecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=external-certificate.io,resources=importcertificatesecrets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;patch;delete

// Reconcile reads that state of the cluster for a ImportCertificateSecret object and makes changes based on the state read
// and what is in the ImportCertificateSecret.Spec
// Terminations are handled by the finalizer, which is added to the ImportCertificateSecret object
// when the object is created and removed when the object is deleted
// Termination is retried with exponential backoff
// The reconciliation loop is stopped when the Terminal condition is set
// Normal reconcile loop with error handling is stopped when the ProcessingFailed condition is set
// The reconciliation loop is requeued with a delay when the Processed condition is set with an exponential backoff

func (r *ImportCertificateSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Ctx = ctx
	log := log.FromContext(r.Ctx)
	log.Info("Starting reconciliation", "namespace", req.Namespace, "name", req.Name)
	r.ImportCertificateSecret = &certdistributionv1alpha1.ImportCertificateSecret{}

	// Check if the CRD is being deleted
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := r.Get(ctx, types.NamespacedName{Name: IMPORT_CERT_SECRET_CRD}, crd); err != nil {
		if errors.IsNotFound(err) {
			log.Info("CRD not found, skipping reconciliation")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !crd.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("CRD is being deleted, removing finalizers from custom resources")
		return ctrl.Result{}, RemoveFinalizersFromAllCustomResources(r.Ctx, r.Client, &certdistributionv1alpha1.ImportCertificateSecretList{})
	}

	// Check if the request is for an ImportCertificateSecret
	err := r.Get(ctx, req.NamespacedName, r.ImportCertificateSecret)
	if err == nil {
		return r.reconcileImportCertificateSecret(req)
	}

	// Check if the request is for a Secret
	secret := &corev1.Secret{}
	err = r.Get(r.Ctx, req.NamespacedName, secret)
	if err == nil {
		return r.reconcileSecret(req, secret)
	}

	// If the Secret is not found, it might be a delete event
	if errors.IsNotFound(err) {
		log.Info("Secret not found, might be a delete event", "namespace", req.Namespace, "name", req.Name)
		return r.reconcileSecret(req, nil)
	}

	// If neither ImportCertificateSecret nor Secret is found, return an error
	log.Error(err, "Failed to get resource")
	return ctrl.Result{}, err
}

// Reconcile logic for ImportCertificateSecret
func (r *ImportCertificateSecretReconciler) reconcileImportCertificateSecret(req ctrl.Request) (ctrl.Result, error) {
	var err error
	log := log.FromContext(r.Ctx)
	log.Info("Reconciling ImportCertificateSecret", "namespace", req.Namespace, "name", req.Name)

	if ConditionExists(r.ImportCertificateSecret.Status.Conditions, metav1.Condition{
		Type:   "Terminal",
		Status: metav1.ConditionTrue,
	}) {
		log.Info("Terminal condition set, stopping reconciliation", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, nil
	}

	if r.ImportCertificateSecret.Status.RetryCount >= maxRetries {
		log.Error(fmt.Errorf("max retries reached"), "Giving up on reconciliation", "name", req.Name, "namespace", req.Namespace)
		return r.updateStatusWithCondition(metav1.Condition{
			Type:    "Terminal",
			Status:  metav1.ConditionTrue,
			Reason:  "MaxRetriesReached",
			Message: "Maximum retry limit reached, stopping reconciliation",
		})
	}

	// Set the Kubernetes client
	r.KubeClient, err = SetKubeClient()
	if err != nil {
		return ctrl.Result{}, err
	}

	var serviceAccount corev1.ServiceAccount
	if err := r.Get(r.Ctx, types.NamespacedName{Name: r.ImportCertificateSecret.Spec.AzureKV.ServiceAccountRef.Name, Namespace: req.Namespace}, &serviceAccount); err != nil {
		log.Error(err, "unable to fetch ServiceAccount")
		return ctrl.Result{}, err
	}

	// Create an authorizer MSAL token for the service account based on annotations
	azureauth := &azureauth.WorkloadID{
		Ctx:            r.Ctx,
		KubeClient:     r.KubeClient,
		ServiceAccount: serviceAccount,
		SaAudiences:    r.ImportCertificateSecret.Spec.AzureKV.ServiceAccountRef.Audiences,
		TokenProvider:  azureauth.NewTokenProvider,
	}
	err = azureauth.AuthorizerForWorkloadIdentity()
	if err != nil {
		log.Error(err, "failed to create authorizer")
		r.ImportCertificateSecret.Status.RetryCount++
		delay := baseDelay * time.Duration(1<<r.ImportCertificateSecret.Status.RetryCount)
		if delay > maxDuration {
			delay = maxDuration
		}
		log.Info("Retrying MSAL auth", "retryCount", r.ImportCertificateSecret.Status.RetryCount, "nextRetryIn", delay)
		_, updateErr := r.updateStatusWithCondition(metav1.Condition{
			Type:    "MSALAuth",
			Status:  metav1.ConditionFalse,
			Reason:  "MSALAuthFailed",
			Message: fmt.Sprintf("Retry %d: %s. Next retry in %s", r.ImportCertificateSecret.Status.RetryCount, err.Error(), delay),
		})
		if updateErr != nil {
			return ctrl.Result{RequeueAfter: delay}, updateErr
		}
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	r.AkvClient = &akv.KeyVault{
		Ctx:      r.Ctx,
		Client:   akv.NewAzureKeyVaultClient(azureauth.Authorizer),
		VaultUrl: r.ImportCertificateSecret.Spec.AzureKV.VaultUrl,
	}

	destinationNamespace := req.Namespace
	if r.ImportCertificateSecret.Spec.AzureKV.SecretNamespace != "" {
		log.Info("Using SecretNamespace from ImportCertificateSecret", "SecretNamespace", r.ImportCertificateSecret.Spec.AzureKV.SecretNamespace)
		destinationNamespace = r.ImportCertificateSecret.Spec.AzureKV.SecretNamespace
	}

	if r.ImportCertificateSecret.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.addFinalizer(); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if ContainsString(r.ImportCertificateSecret.ObjectMeta.Finalizers, FINALIZER) {
			if r.ImportCertificateSecret.Status.RetryCount >= maxRetries {
				log.Error(fmt.Errorf("max retries reached"), "Giving up on termination", "name", req.Name, "namespace", req.Namespace)
				return r.updateStatusWithCondition(metav1.Condition{
					Type:    "Terminal",
					Status:  metav1.ConditionTrue,
					Reason:  "MaxRetriesReached",
					Message: "Maximum retry limit reached, stopping reconciliation",
				})
			}

			if err := r.ProcessImportSecret(r.ImportCertificateSecret.Spec.AzureKV, destinationNamespace, "delete"); err != nil {
				log.Error(err, "unable to delete secret")
				r.ImportCertificateSecret.Status.RetryCount++
				delay := baseDelayDelete * (1 << r.ImportCertificateSecret.Status.RetryCount)
				if delay > maxDuration {
					delay = maxDuration
				}
				log.Info("Retrying deletion", "retryCount", r.ImportCertificateSecret.Status.RetryCount, "nextRetryIn", delay)
				_, updateErr := r.updateStatusWithCondition(metav1.Condition{
					Type:    "Processed",
					Status:  metav1.ConditionFalse,
					Reason:  "CertificateProcessingFailed",
					Message: fmt.Sprintf("Certificate secret sync failed this is retry %d: %s. Next retry in %s", r.ImportCertificateSecret.Status.RetryCount, err.Error(), delay),
				})
				if updateErr != nil {
					return ctrl.Result{RequeueAfter: delay}, updateErr
				}
				return ctrl.Result{RequeueAfter: delay}, nil
			}
			if err := r.removeFinalizer(); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if err := r.ProcessImportSecret(r.ImportCertificateSecret.Spec.AzureKV, destinationNamespace, "create"); err != nil {
		log.Error(err, "unable to create secret")
		r.ImportCertificateSecret.Status.RetryCount++
		delay := baseDelay * (1 << r.ImportCertificateSecret.Status.RetryCount)
		if delay > maxDuration {
			delay = maxDuration
		}
		log.Info("Retrying creation", "retryCount", r.ImportCertificateSecret.Status.RetryCount, "nextRetryIn", delay)
		_, updateErr := r.updateStatusWithCondition(metav1.Condition{
			Type:    "Processed",
			Status:  metav1.ConditionFalse,
			Reason:  "CertificateProcessingFailed",
			Message: fmt.Sprintf("Certificate secret sync failed this is retry %d: %s. Next retry in %s", r.ImportCertificateSecret.Status.RetryCount, err.Error(), delay),
		})
		if updateErr != nil {
			return ctrl.Result{RequeueAfter: delay}, updateErr
		}
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	r.ImportCertificateSecret.Status.RetryCount = 0
	result, updateErr := r.updateStatusWithCondition(metav1.Condition{
		Type:    "Processed",
		Status:  metav1.ConditionTrue,
		Reason:  "CertificateProcessingSucceeded",
		Message: "Secret sync processed successfully",
	})
	if updateErr != nil {
		return ctrl.Result{}, updateErr
	}
	return result, nil
}

// Reconcile logic for Secret
func (r *ImportCertificateSecretReconciler) reconcileSecret(req ctrl.Request, secret *corev1.Secret) (ctrl.Result, error) {
	log := log.FromContext(r.Ctx)
	log.Info("Reconciling Secret", "namespace", req.Namespace, "name", req.Name)

	// Fetch the corresponding ImportCertificateSecret
	importCertSecretList := &certdistributionv1alpha1.ImportCertificateSecretList{}
	if err := r.List(r.Ctx, importCertSecretList, &client.ListOptions{}); err != nil {
		log.Error(err, "Failed to list ImportCertificateSecrets")
		return ctrl.Result{}, err
	}

	var relatedImportCertSecret *certdistributionv1alpha1.ImportCertificateSecret
	for _, importCertSecret := range importCertSecretList.Items {
		secretNamespace := importCertSecret.Namespace
		if importCertSecret.Spec.AzureKV.SecretNamespace != "" {
			secretNamespace = importCertSecret.Spec.AzureKV.SecretNamespace
		}

		// Check if secret is nil before accessing its fields
		if secret != nil && secret.Namespace == secretNamespace {
			for _, certRef := range importCertSecret.Spec.AzureKV.CertificateSecretRef {
				if secret.Name == certRef.SecretName {
					currentImportCertSecret := importCertSecret
					relatedImportCertSecret = &currentImportCertSecret
					break
				}
			}
		} else if secret == nil {
			// Handle the case where secret is nil (delete event)
			for _, certRef := range importCertSecret.Spec.AzureKV.CertificateSecretRef {
				if req.Name == certRef.SecretName && secretNamespace == req.Namespace {
					currentImportCertSecret := importCertSecret
					relatedImportCertSecret = &currentImportCertSecret
					break
				}
			}
		}

		if relatedImportCertSecret != nil {
			break
		}
	}

	if relatedImportCertSecret == nil {
		log.Info("No matching ImportCertificateSecret found for Secret", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, nil
	}
	log.Info("Found matching ImportCertificateSecret", "namespace", relatedImportCertSecret.Namespace, "name", relatedImportCertSecret.Name)

	// Add an annotation to the ImportCertificateSecret to indicate that the secret has changed causing a requeue of ImportCertificateSecret reconciliation
	err := r.AddAnnotationToImportCertificateSecret(relatedImportCertSecret.Namespace, relatedImportCertSecret.Name, SECRET_CHG_ANNOTATION, time.Now().Format("2006-01-02T15:04:05"))
	if err != nil {
		log.Error(err, "Failed to add annotation to ImportCertificateSecret")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// AddAnnotationToImportCertificateSecret adds an annotation to the specified ImportCertificateSecret object
func (r *ImportCertificateSecretReconciler) AddAnnotationToImportCertificateSecret(namespace, name, key, value string) error {
	// Fetch the ImportCertificateSecret object
	importCertSecret := &certdistributionv1alpha1.ImportCertificateSecret{}
	if err := r.Get(r.Ctx, client.ObjectKey{Namespace: namespace, Name: name}, importCertSecret); err != nil {
		return fmt.Errorf("failed to get ImportCertificateSecret: %w", err)
	}

	// Add or update the annotation
	if importCertSecret.Annotations == nil {
		importCertSecret.Annotations = make(map[string]string)
	}
	importCertSecret.Annotations[key] = value

	// Update the ImportCertificateSecret object in the Kubernetes cluster
	if err := r.Update(r.Ctx, importCertSecret); err != nil {
		return fmt.Errorf("failed to update ImportCertificateSecret: %w", err)
	}

	return nil
}

// ProcessImportSecret processes the Kubernetes secret based on the type
func (r *ImportCertificateSecretReconciler) ProcessImportSecret(importCertSecret certdistributionv1alpha1.ImportAzureKVProvider, namespace string, action string) error {
	log := log.FromContext(r.Ctx)
	r.AkvClient.Secret = &akv.Secret{}
	// Group keys by secretName
	secretGroups := make(map[string]map[string]string)
	for _, certRef := range r.ImportCertificateSecret.Spec.AzureKV.CertificateSecretRef {
		if _, exists := secretGroups[certRef.SecretName]; !exists {
			secretGroups[certRef.SecretName] = make(map[string]string)
		}
		secretGroups[certRef.SecretName][certRef.SecretKey] = certRef.KVSecretName
	}

	for secretName, keys := range secretGroups {
		namespacedSecret := types.NamespacedName{
			Namespace: namespace,
			Name:      secretName,
		}
		secretObject := &corev1.Secret{}
		if err := r.Get(r.Ctx, namespacedSecret, secretObject); err != nil {
			log.Info("Unable to fetch secret, assuming secret not yet created", "secretName", secretName)
		}
		keysMap := make(map[string][]byte)
		switch action {
		case "create":
			for secretKey, kvSecretName := range keys {
				log.Info("msg", "kvSecretName", kvSecretName)
				r.AkvClient.Secret.Name = kvSecretName
				// Check if the secret exists in Key Vault
				err := r.AkvClient.CheckKVSecretExistence()
				if err != nil {
					return err
				}

				if r.AkvClient.Secret.SecretBundle.Value == nil {
					log.Info("Secret not found in Key Vault", "secretNameNotFound", kvSecretName)
					continue
				}

				if IsBase64Encoded([]byte(*r.AkvClient.Secret.SecretBundle.Value)) {
					decodedValue, err := base64.StdEncoding.DecodeString(*r.AkvClient.Secret.SecretBundle.Value)
					if err != nil {
						log.Error(err, "Failed to decode base64 string: %v", err)
						return err
					}
					decodedValueStr := string(decodedValue)
					r.AkvClient.Secret.SecretBundle.Value = &decodedValueStr
				}
				keysMap[secretKey] = []byte(*r.AkvClient.Secret.SecretBundle.Value)
				log.Info("Successfully fetched secret from Key Vault", "kvSecretFetch", kvSecretName)
			}

			// Ensure minimum keys are present and prepopulate with empty strings if missing. k8s secret of type tls require tls
			if _, ok := keysMap["tls.key"]; !ok {
				log.Info("Prepopulating missing tls.key with empty string", "missingSecretKey", secretName)
				keysMap["tls.key"] = []byte("")
			}
			if _, ok := keysMap["tls.crt"]; !ok {
				log.Info("Prepopulating missing tls.crt with empty string", "missingSecretKey", secretName)
				keysMap["tls.crt"] = []byte("")
			}

			// Create or patch the Kubernetes secret with all keys
			if err := CreateOrPatchK8sSecret(r.Ctx, r.KubeClient, namespacedSecret.Name, keysMap, namespacedSecret.Namespace); err != nil {
				log.Error(err, "unable to create or patch destination Kubernetes secret")
				return err
			}
			log.Info("Successfully created or patched destination Kubernetes secret", "secretUpdate", namespacedSecret.Name)
		case "delete":
			// Retry deleting the secret with exponential backoff
			initialDelay := 2 * time.Second
			maxDelay := 30 * time.Second

			for i := 0; i < maxRetries; i++ {
				err := r.Delete(r.Ctx, secretObject)
				if err == nil {
					log.Info("Successfully deleted k8s secret", "name", namespacedSecret.Name, "namespace", namespacedSecret.Namespace)
					break
				}
				if errors.IsNotFound(err) {
					log.Info("Secret not found, no need to delete", "name", namespacedSecret.Name, "namespace", namespacedSecret.Namespace)
					break
				}
				log.Info("Failed to delete secret from K8s, retrying...", "attempt", i+1, "error", err)
				time.Sleep(time.Duration(math.Min(float64(maxDelay), float64(initialDelay)*math.Pow(2, float64(i)))))
			}
		default:
			// Return an error if the action is unsupported
			return fmt.Errorf("unsupported action: %s", action)
		}
	}
	return nil
}

func (r *ImportCertificateSecretReconciler) updateStatusWithCondition(condition metav1.Condition) (ctrl.Result, error) {
	log := log.FromContext(r.Ctx)
	condition.LastTransitionTime = metav1.Now()

	// Find the existing condition
	existingCondition := findCondition(r.ImportCertificateSecret.Status.Conditions, condition.Type)
	if existingCondition != nil {
		// Update the existing condition
		existingCondition.Status = condition.Status
		existingCondition.Reason = condition.Reason
		existingCondition.Message = condition.Message
		existingCondition.LastTransitionTime = condition.LastTransitionTime
	} else {
		// Append the new condition
		r.ImportCertificateSecret.Status.Conditions = append(r.ImportCertificateSecret.Status.Conditions, condition)
	}

	if err := r.Status().Update(r.Ctx, r.ImportCertificateSecret); err != nil {
		log.Error(err, "unable to update ImportCertificateSecret status")
		if getErr := r.Get(r.Ctx, types.NamespacedName{Name: r.ImportCertificateSecret.Name, Namespace: r.ImportCertificateSecret.Namespace}, r.ImportCertificateSecret); getErr == nil {
			if retryUpdateErr := r.Status().Update(r.Ctx, r.ImportCertificateSecret); retryUpdateErr != nil {
				log.Error(retryUpdateErr, "retry update failed")
				return ctrl.Result{}, retryUpdateErr
			}
		} else {
			log.Error(getErr, "unable to fetch latest ImportCertificateSecret")
			return ctrl.Result{}, getErr
		}
	}

	// Create an event
	r.createEvent(condition)

	log.Info("Successfully updated ImportCertificateSecret status", "namespace", r.ImportCertificateSecret.Namespace, "name", r.ImportCertificateSecret.Name)
	return ctrl.Result{RequeueAfter: time.Duration(r.ImportCertificateSecret.Spec.AzureKV.ScanInterval) * time.Minute}, nil
}

// createEvent creates a Kubernetes event based on the condition
func (r *ImportCertificateSecretReconciler) createEvent(condition metav1.Condition) {
	log := log.FromContext(r.Ctx)
	log.Info("Creating event", "reason", condition.Reason, "message", condition.Message)
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: r.ImportCertificateSecret.Name + "-",
			Namespace:    r.ImportCertificateSecret.Namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       "ImportCertificateSecret",
			Namespace:  r.ImportCertificateSecret.Namespace,
			Name:       r.ImportCertificateSecret.Name,
			UID:        r.ImportCertificateSecret.UID,
			APIVersion: "external-certificate.io/v1alpha1",
		},
		Reason:  condition.Reason,
		Message: condition.Message,
		Type:    string(condition.Status),
		Source: corev1.EventSource{
			Component: "importcertificatesecret-controller",
		},
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Count:          1,
	}

	if err := r.Create(r.Ctx, event); err != nil {
		log.Error(err, "unable to create event")
	}
}

func (r *ImportCertificateSecretReconciler) removeFinalizer() error {
	r.ImportCertificateSecret.ObjectMeta.Finalizers = RemoveString(r.ImportCertificateSecret.ObjectMeta.Finalizers, FINALIZER)
	if err := r.Update(r.Ctx, r.ImportCertificateSecret); err != nil {
		return err
	}
	return nil
}

func (r *ImportCertificateSecretReconciler) addFinalizer() error {
	if !ContainsString(r.ImportCertificateSecret.ObjectMeta.Finalizers, FINALIZER) {
		r.ImportCertificateSecret.ObjectMeta.Finalizers = append(r.ImportCertificateSecret.ObjectMeta.Finalizers, FINALIZER)
		if err := r.Update(r.Ctx, r.ImportCertificateSecret); err != nil {
			return err
		}
	}
	return nil
}

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
