package controller

import (
	"context"
	"encoding/base64"
	stdErrors "errors"
	"fmt"
	"time"

	"github.com/Azure/go-autorest/autorest/date"
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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"

	certdistributionv1alpha1 "github.com/gigabytte/external-certificate-operator/api/v1alpha1"
	"github.com/gigabytte/external-certificate-operator/internal/log"
	"github.com/gigabytte/external-certificate-operator/internal/openssl"
	akv "github.com/gigabytte/external-certificate-operator/internal/providers/azure/akv"
	azureauth "github.com/gigabytte/external-certificate-operator/internal/providers/azure/auth"
)

// ExportCertificateSecretReconciler reconciles an ExportCertificateSecret object
type ExportCertificateSecretReconciler struct {
	client.Client
	AkvClient               *akv.KeyVault
	KubeClient              kubernetes.Interface
	Scheme                  *runtime.Scheme
	Ctx                     context.Context
	OpenSSLRunners          *openssl.OpenSSLRunners
	ExportCertificateSecret *certdistributionv1alpha1.ExportCertificateSecret
}

//+kubebuilder:rbac:groups=external-certificate.io,resources=exportcertificatesecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=external-certificate.io,resources=exportcertificatesecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=external-certificate.io,resources=exportcertificatesecrets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=serviceaccounts/token,verbs=create

// Reconcile reads that state of the cluster for a ExportCertificateSecret object and makes changes based on the state read
// and what is in the ExportCertificateSecret.Spec
// Terminations are handled by the finalizer, which is added to the ExportCertificateSecret object
// when the object is created and removed when the object is deleted
// Termination is retried with exponential backoff
// The reconciliation loop is stopped when the Terminal condition is set
// Normal reconcile loop with error handling is stopped when the ProcessingFailed condition is set
// The reconciliation loop is requeued with a delay when the Processed condition is set with an exponential backoff

func (r *ExportCertificateSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Ctx = ctx
	log := log.FromContext(r.Ctx)
	log.Info("Starting reconciliation", "namespace", req.Namespace, "name", req.Name)

	// Check if the CRD is being deleted
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := r.Get(ctx, types.NamespacedName{Name: EXPORT_CERT_SECRET_CRD}, crd); err != nil {
		if errors.IsNotFound(err) {
			log.Info("CRD not found, skipping reconciliation")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !crd.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("CRD is being deleted, removing finalizers from custom resources")
		return ctrl.Result{}, RemoveFinalizersFromAllCustomResources(r.Ctx, r.Client, &certdistributionv1alpha1.ExportCertificateSecret{})
	}

	r.ExportCertificateSecret = &certdistributionv1alpha1.ExportCertificateSecret{}
	err := r.Get(r.Ctx, req.NamespacedName, r.ExportCertificateSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if ConditionExists(r.ExportCertificateSecret.Status.Conditions, metav1.Condition{
		Type:   "Terminal",
		Status: metav1.ConditionTrue,
	}) {
		log.Info("Terminal condition set, stopping reconciliation", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, nil
	}

	if r.ExportCertificateSecret.Status.RetryCount >= maxRetries {
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
	if err := r.Get(r.Ctx, types.NamespacedName{Name: r.ExportCertificateSecret.Spec.AzureKV.ServiceAccountRef.Name, Namespace: req.Namespace}, &serviceAccount); err != nil {
		log.Error(err, "unable to fetch ServiceAccount")
		return ctrl.Result{}, err
	}

	// Create an authorizer MSAL token for the service account based on annotations
	azureauth := &azureauth.WorkloadID{
		Ctx:            r.Ctx,
		KubeClient:     r.KubeClient,
		ServiceAccount: serviceAccount,
		SaAudiences:    r.ExportCertificateSecret.Spec.AzureKV.ServiceAccountRef.Audiences,
		TokenProvider:  azureauth.NewTokenProvider,
	}
	err = azureauth.AuthorizerForWorkloadIdentity()
	if err != nil {
		log.Error(err, "failed to create authorizer")
		r.ExportCertificateSecret.Status.RetryCount++
		delay := baseDelay * time.Duration(1<<r.ExportCertificateSecret.Status.RetryCount)
		if delay > maxDuration {
			delay = maxDuration
		}
		log.Info("Retrying MSAL auth", "retryCount", r.ExportCertificateSecret.Status.RetryCount, "nextRetryIn", delay)
		_, updateErr := r.updateStatusWithCondition(metav1.Condition{
			Type:    "MSALAuth",
			Status:  metav1.ConditionFalse,
			Reason:  "MSALAuthFailed",
			Message: fmt.Sprintf("Retry %d: %s. Next retry in %s", r.ExportCertificateSecret.Status.RetryCount, err.Error(), delay),
		})
		if updateErr != nil {
			return ctrl.Result{RequeueAfter: delay}, updateErr
		}
		return ctrl.Result{RequeueAfter: delay}, nil
	}
	r.AkvClient = &akv.KeyVault{
		Ctx:      r.Ctx,
		Client:   akv.NewAzureKeyVaultClient(azureauth.Authorizer),
		VaultUrl: r.ExportCertificateSecret.Spec.AzureKV.VaultUrl,
	}

	if r.ExportCertificateSecret.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.addFinalizer(); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if ContainsString(r.ExportCertificateSecret.ObjectMeta.Finalizers, FINALIZER) {
			if r.ExportCertificateSecret.Status.RetryCount >= maxRetries {
				log.Error(fmt.Errorf("max retries reached"), "Giving up on termination", "name", req.Name, "namespace", req.Namespace)
				return r.updateStatusWithCondition(metav1.Condition{
					Type:    "Terminal",
					Status:  metav1.ConditionTrue,
					Reason:  "MaxRetriesReached",
					Message: "Maximum retry limit reached, stopping reconciliation",
				})
			}
			if err := r.DeleteSecret(nil); err != nil {
				log.Error(err, "unable to delete secret")
				r.ExportCertificateSecret.Status.RetryCount++
				delay := baseDelayDelete * time.Duration(1<<r.ExportCertificateSecret.Status.RetryCount)
				if delay > maxDuration {
					delay = maxDuration
				}
				log.Info("Retrying deletion", "retryCount", r.ExportCertificateSecret.Status.RetryCount, "nextRetryIn", delay)
				_, updateErr := r.updateStatusWithCondition(metav1.Condition{
					Type:    "Processed",
					Status:  metav1.ConditionFalse,
					Reason:  "CertificateProcessingFailed",
					Message: fmt.Sprintf("Certificate secret sync failed this is retry %d: %s. Next retry in %s", r.ExportCertificateSecret.Status.RetryCount, err.Error(), delay),
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

	// Compare current CertificateSecretRef with previous state
	if err := r.compareAndSyncSecrets(req.Namespace); err != nil {
		r.ExportCertificateSecret.Status.RetryCount++
		delay := baseDelay * time.Duration(1<<r.ExportCertificateSecret.Status.RetryCount)
		if delay > maxDuration {
			delay = maxDuration
		}
		log.Info("Retrying creation", "retryCount", r.ExportCertificateSecret.Status.RetryCount, "nextRetryIn", delay)
		_, updateErr := r.updateStatusWithCondition(metav1.Condition{
			Type:    "Processed",
			Status:  metav1.ConditionFalse,
			Reason:  "CertificateProcessingFailed",
			Message: fmt.Sprintf("Certificate secret sync failed this is retry %d: %s. Next retry in %s", r.ExportCertificateSecret.Status.RetryCount, err.Error(), delay),
		})
		if updateErr != nil {
			return ctrl.Result{RequeueAfter: delay}, updateErr
		}
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	r.ExportCertificateSecret.Status.RetryCount = 0
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

func (r *ExportCertificateSecretReconciler) SeedCertificates(namespace string, certRefs []certdistributionv1alpha1.CertificateSecretRef) error {
	log := log.FromContext(r.Ctx)
	r.OpenSSLRunners = openssl.NewOpenSSLRunners(r.Ctx)

	for _, cert := range certRefs {
		namespacedSecret := types.NamespacedName{
			Namespace: namespace,
			Name:      cert.SecretName,
		}
		var k8sSecret corev1.Secret
		if err := r.Get(r.Ctx, namespacedSecret, &k8sSecret); err != nil {
			log.Error(err, "unable to fetch referenced Kubernetes secret")
			return err
		}
		// Check if the secret version exists in Key Vault based on byte comparison
		secretData, ok := k8sSecret.Data[cert.SecretKey]
		if !ok {
			log.Error(nil, "Secret key not found in Kubernetes secret")
			return fmt.Errorf("Secret key not found in Kubernetes secret")
		}
		switch cert.SecretKey {
		case "tls.crt":
			r.OpenSSLRunners.Certificates.TLSCrt = secretData
		case "tls.key":
			r.OpenSSLRunners.Certificates.TLSKey = secretData
		case "ca.crt":
			r.OpenSSLRunners.Certificates.CACrt = secretData
		case "tls-combined.pem":
			r.OpenSSLRunners.Certificates.PemBundle = secretData
		default:
			log.Error(fmt.Errorf("Secret key with value not supported"), "secretKey", cert.SecretKey)
			continue
		}
	}
	if err := r.OpenSSLRunners.ProcessCertificates(); err != nil {
		return err
	}

	return nil
}

// SyncSecret processes the Kubernetes secret based on the type
func (r *ExportCertificateSecretReconciler) SyncSecret(namespace string, certRef []certdistributionv1alpha1.CertificateSecretRef) error {
	log := log.FromContext(r.Ctx)
	var k8sSecret corev1.Secret
	secretAttributes := keyvault.SecretAttributes{}

	if r.OpenSSLRunners.Certificates.CertificateDetails.ExpirationTime != 0 {
		secretAttributes.Expires = func(t date.UnixTime) *date.UnixTime { return &t }(date.NewUnixTimeFromSeconds(r.OpenSSLRunners.Certificates.CertificateDetails.ExpirationTime))
	} else {
		log.Info("Certificate expiration time not found, this is probably due to lack of .crt secret being synced with AKV. See logs for more details")
	}

	for _, cert := range certRef {
		namespacedSecret := types.NamespacedName{
			Namespace: namespace,
			Name:      cert.SecretName,
		}
		if err := r.Get(r.Ctx, namespacedSecret, &k8sSecret); err != nil {
			log.Error(err, "unable to fetch referenced Kubernetes secret")
			return err
		}
		secretName := cert.KVSecretName
		// Check if the secret version exists in Key Vault based on byte comparison
		secretData, ok := k8sSecret.Data[cert.SecretKey]
		if !ok {
			log.Error(nil, "Secret key not found in Kubernetes secret")
			return fmt.Errorf("Secret key not found in Kubernetes secret")
		}
		r.AkvClient.Secret = &akv.Secret{
			Name:       secretName,
			Value:      string(secretData),
			Attributes: secretAttributes,
		}
		// Check if the secret exists in Key Vault
		err := r.AkvClient.CheckKVSecretExistence()
		if err != nil {
			return err
		}
		if cert.Type == "crt" {
			switch cert.Templating {
			case BASE64EN_TEMPLATING:
				r.AkvClient.Secret.Value = base64.StdEncoding.EncodeToString([]byte(r.AkvClient.Secret.Value))
			}
			if r.AkvClient.Secret.SecretBundle.Value != nil {
				// Check if the secret version exists in Key Vault based on byte comparison
				ok, _ := ByteCompare(r.Ctx, []byte(r.AkvClient.Secret.Value), *r.AkvClient.Secret.SecretBundle.Value)
				if ok {
					// Continue to next iteration since the current secret matches kv
					log.Info("Secret key matches that of in Key Vault", "secretKeyMatch", cert.SecretKey)
					continue
				}
			}
			// Push to Key Vault
			err := r.AkvClient.PushSecretToKeyVault()
			if err != nil {
				log.Error(err, "unable to push secret to Key Vault", "secretName", secretName)
				return err
			}
			log.Info("Successfully pushed secret to Key Vault", "secretKVPush", secretName)
			continue
		}
		if cert.Type == "pem" {
			switch cert.Templating {
			case PEMTOPFX_TEMPLATING:
				r.AkvClient.Secret.Value = base64.StdEncoding.EncodeToString(r.OpenSSLRunners.Certificates.ConvertedFormats.PFXCertificate)
			case BASE64EN_TEMPLATING:
				r.AkvClient.Secret.Value = base64.StdEncoding.EncodeToString(r.OpenSSLRunners.Certificates.PemBundle)
			}
			if r.AkvClient.Secret.SecretBundle.Value != nil {
				// Check if the secret version exists in Key Vault based on byte comparison
				ok, _ := ByteCompare(r.Ctx, []byte(r.AkvClient.Secret.Value), *r.AkvClient.Secret.SecretBundle.Value)
				if ok {
					// Continue to next iteration since the current secret matches kv
					log.Info("Secret key matches that of in Key Vault", "secretKeyMatch", cert.SecretKey)
					continue
				}
			}
			// Push to Key Vault
			err := r.AkvClient.PushCertificateToKeyVault()
			if err != nil {
				log.Error(err, "unable to push pem to Key Vault")
				return err
			}
		}
	}

	return nil
}

// DeleteSecret deletes the secrets from Key Vault sequentially
func (r *ExportCertificateSecretReconciler) DeleteSecret(certRef []certdistributionv1alpha1.CertificateSecretRef) error {
	if certRef == nil {
		certRef = r.ExportCertificateSecret.Spec.AzureKV.CertificateSecretRef
	}

	var errList []error

	for _, cert := range certRef {
		secretName := cert.KVSecretName
		r.AkvClient.Secret = &akv.Secret{
			Name: secretName,
		}
		if cert.Type == "crt" {
			err := r.AkvClient.DeleteSecretFromKeyVault()
			if err != nil {
				errList = append(errList, err)
				continue
			}
			if r.ExportCertificateSecret.Spec.AzureKV.OnDeletePurge {
				if err := r.AkvClient.PurgeDeletedSecretFromKeyVault(); err != nil {
					errList = append(errList, err)
					continue
				}
			}
		}
		if cert.Type == "pem" {
			err := r.AkvClient.DeleteCertificateFromKeyVault()
			if err != nil {
				errList = append(errList, err)
				continue
			}
			if r.ExportCertificateSecret.Spec.AzureKV.OnDeletePurge {
				if err := r.AkvClient.PurgeDeletedCertificateFromKeyVault(); err != nil {
					errList = append(errList, err)
					continue
				}
			}
		}
	}

	if len(errList) > 0 {
		return stdErrors.New("one or more errors occurred during deletion")
	}

	return nil
}

// compareAndSyncSecrets compares the current CertificateSecretRef with the previous state and syncs the secrets
func (r *ExportCertificateSecretReconciler) compareAndSyncSecrets(namespace string) error {
	log := log.FromContext(r.Ctx)
	// Get the previous state from the status
	previousSecretRefs := r.ExportCertificateSecret.Status.PreviousSecretRefs
	// Compare with the current state
	currentSecretRefs := r.ExportCertificateSecret.Spec.AzureKV.CertificateSecretRef

	log.Info("previousSecretRefs", "previousSecretRefs", previousSecretRefs)
	log.Info("currentSecretRefs", "currentSecretRefs", currentSecretRefs)

	// Seed certificates
	if err := r.SeedCertificates(namespace, currentSecretRefs); err != nil {
		return err
	}
	if previousSecretRefs == nil {
		// Sync secrets
		if err := r.SyncSecret(namespace, currentSecretRefs); err != nil {
			return err
		}
		r.ExportCertificateSecret.Status.PreviousSecretRefs = currentSecretRefs
		return nil
	}
	// Find differences
	added, removed := findDifferences(previousSecretRefs, currentSecretRefs)

	// Log differences
	log.Info("Secrets to be created in KV", "secretDiff", added)
	log.Info("Secrets to be removed from KV", "secretDiff", removed)
	if len(added) > 0 {
		// Sync added secrets
		if err := r.SyncSecret(namespace, added); err != nil {
			return err
		}
	}
	if len(removed) > 0 {
		// Sync removed secrets
		if err := r.DeleteSecret(removed); err != nil {
			return err
		}
	}
	// Update the previous state in the status
	r.ExportCertificateSecret.Status.PreviousSecretRefs = currentSecretRefs

	return nil
}

// findDifferences finds the differences between two lists of CertificateSecretRef
func findDifferences(oldRefs, newRefs []certdistributionv1alpha1.CertificateSecretRef) (added, removed []certdistributionv1alpha1.CertificateSecretRef) {
	oldMap := make(map[string]certdistributionv1alpha1.CertificateSecretRef)
	newMap := make(map[string]certdistributionv1alpha1.CertificateSecretRef)

	for _, ref := range oldRefs {
		oldMap[ref.KVSecretName] = ref
	}

	for _, ref := range newRefs {
		newMap[ref.KVSecretName] = ref
	}

	for name, ref := range newMap {
		if _, exists := oldMap[name]; !exists {
			added = append(added, ref)
		}
	}

	for name, ref := range oldMap {
		if _, exists := newMap[name]; !exists {
			removed = append(removed, ref)
		}
	}

	return added, removed
}

func (r *ExportCertificateSecretReconciler) updateStatusWithCondition(condition metav1.Condition) (ctrl.Result, error) {
	log := log.FromContext(r.Ctx)
	condition.LastTransitionTime = metav1.Now()

	// Find the existing condition
	existingCondition := findCondition(r.ExportCertificateSecret.Status.Conditions, condition.Type)
	if existingCondition != nil {
		// Update the existing condition
		existingCondition.Status = condition.Status
		existingCondition.Reason = condition.Reason
		existingCondition.Message = condition.Message
		existingCondition.LastTransitionTime = condition.LastTransitionTime
	} else {
		// Append the new condition
		r.ExportCertificateSecret.Status.Conditions = append(r.ExportCertificateSecret.Status.Conditions, condition)
	}

	if err := r.Status().Update(r.Ctx, r.ExportCertificateSecret); err != nil {
		log.Error(err, "unable to update ExportCertificateSecret status")
		if getErr := r.Get(r.Ctx, types.NamespacedName{Name: r.ExportCertificateSecret.Name, Namespace: r.ExportCertificateSecret.Namespace}, r.ExportCertificateSecret); getErr == nil {
			if retryUpdateErr := r.Status().Update(r.Ctx, r.ExportCertificateSecret); retryUpdateErr != nil {
				log.Error(retryUpdateErr, "retry update failed")
				return ctrl.Result{}, retryUpdateErr
			}
		} else {
			log.Error(getErr, "unable to fetch latest ExportCertificateSecret")
			return ctrl.Result{}, getErr
		}
	}

	// Create an event
	r.createEvent(condition)

	log.Info("Successfully updated ExportCertificateSecret status", "namespace", r.ExportCertificateSecret.Namespace, "name", r.ExportCertificateSecret.Name)
	return ctrl.Result{RequeueAfter: time.Duration(r.ExportCertificateSecret.Spec.AzureKV.ScanInterval) * time.Minute}, nil
}

// createEvent creates a Kubernetes event based on the condition
func (r *ExportCertificateSecretReconciler) createEvent(condition metav1.Condition) {
	log := log.FromContext(r.Ctx)
	log.Info("Creating event", "reason", condition.Reason, "message", condition.Message)
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: r.ExportCertificateSecret.Name + "-",
			Namespace:    r.ExportCertificateSecret.Namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       "ExportCertificateSecret",
			Namespace:  r.ExportCertificateSecret.Namespace,
			Name:       r.ExportCertificateSecret.Name,
			UID:        r.ExportCertificateSecret.UID,
			APIVersion: "external-certificate.io/v1alpha1",
		},
		Reason:  condition.Reason,
		Message: condition.Message,
		Type:    string(condition.Status),
		Source: corev1.EventSource{
			Component: "exportcertificatesecret-controller",
		},
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Count:          1,
	}

	if err := r.Create(r.Ctx, event); err != nil {
		log.Error(err, "unable to create event")
	}
}

// removeFinalizer removes the finalizer from the ExportCertificateSecret object
func (r *ExportCertificateSecretReconciler) removeFinalizer() error {
	r.ExportCertificateSecret.ObjectMeta.Finalizers = RemoveString(r.ExportCertificateSecret.ObjectMeta.Finalizers, FINALIZER)
	if err := r.Update(r.Ctx, r.ExportCertificateSecret); err != nil {
		return err
	}
	return nil
}

// addFinalizer adds the finalizer to the ExportCertificateSecret object
func (r *ExportCertificateSecretReconciler) addFinalizer() error {
	if !ContainsString(r.ExportCertificateSecret.ObjectMeta.Finalizers, FINALIZER) {
		r.ExportCertificateSecret.ObjectMeta.Finalizers = append(r.ExportCertificateSecret.ObjectMeta.Finalizers, FINALIZER)
		if err := r.Update(r.Ctx, r.ExportCertificateSecret); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExportCertificateSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Register the CRD type in the scheme
	if err := apiextensionsv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	namePredicate := predicate.NewPredicateFuncs(func(object client.Object) bool {
		if secret, ok := object.(*corev1.Secret); ok {
			exportCertSecretList := &certdistributionv1alpha1.ExportCertificateSecretList{}
			if err := r.List(context.TODO(), exportCertSecretList, &client.ListOptions{
				Namespace: secret.Namespace,
			}); err != nil {
				return false
			}

			for _, exportCertSecret := range exportCertSecretList.Items {
				for _, certRef := range exportCertSecret.Spec.AzureKV.CertificateSecretRef {
					if secret.Name == certRef.SecretName {
						return true
					}
				}
			}
		}
		return false
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&certdistributionv1alpha1.ExportCertificateSecret{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}). // Ignore status updates to prevent unnecessary reconciliations
		Watches(
			&corev1.Secret{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(namePredicate),
		).
		Complete(r)
}
