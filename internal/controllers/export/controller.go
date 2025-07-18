package exportcertctrl

import (
	"context"
	"encoding/base64"
	stdErrors "errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/Azure/go-autorest/autorest/date"
	certdistributionv1alpha1 "github.com/gigabytte/external-certificate-operator/api/v1alpha1"
	"github.com/gigabytte/external-certificate-operator/internal/shared/log"
	"github.com/gigabytte/external-certificate-operator/internal/shared/openssl"
	akv "github.com/gigabytte/external-certificate-operator/internal/shared/providers/azure/akv"
	azureauth "github.com/gigabytte/external-certificate-operator/internal/shared/providers/azure/auth"
	"github.com/gigabytte/external-certificate-operator/internal/shared/utils"
	"github.com/gigabytte/external-certificate-operator/internal/shared/vars"
	logr "github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExportCertificateSecretReconciler reconciles an ExportCertificateSecret object
type ExportCertificateSecretReconciler struct {
	client.Client
	AkvClient               *akv.KeyVault
	KubeClient              kubernetes.Interface
	Scheme                  *runtime.Scheme
	Ctx                     context.Context
	Logger                  logr.Logger
	OpenSSLRunners          *openssl.OpenSSLRunners
	ExportCertificateSecret *certdistributionv1alpha1.ExportCertificateSecret
	eventHandler            *utils.EventHandler
}

func NewExportCertificateSecretReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
) *ExportCertificateSecretReconciler {
	opts := utils.DefaultOptions(recorder, client)

	return &ExportCertificateSecretReconciler{
		Client:       client,
		eventHandler: utils.NewEventHandler(opts),
	}
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
	// Init logger and context
	r.Ctx = log.IntoContext(ctx, log.FromContext(ctx))
	logger := log.FromContext(r.Ctx)
	r.Logger = logger
	r.Logger.Info("Starting reconciliation")
	var err error

	if err := r.preChecks(req); err != nil {
		return ctrl.Result{}, err
	}

	// Check if object is  being deleted if so trigger finalizers
	if !r.ExportCertificateSecret.DeletionTimestamp.IsZero() {
		r.Logger.Info("ExportCertSecret object is being deleted, removing finalizers from custom resources")
		// Best effort removal of secrets from Key Vault
		// nolint:errcheck
		r.DeleteSecret(r.ExportCertificateSecret.Spec.AzureKV.CertificateSecretRef)

		return ctrl.Result{}, utils.RemoveFinalizers(r.Ctx, r.Logger, r.Client, &certdistributionv1alpha1.ExportCertificateSecret{})
	}
	// Increment the retry count and update the status if not above the max retries
	if r.ExportCertificateSecret.Status.RetryCount >= vars.MaxRetries {
		return r.eventHandler.ReturnEvent(
			r.Ctx,
			r.ExportCertificateSecret,
			r.ExportCertificateSecret.Status.RetryCount,
			stdErrors.New("max retries exceeded"),
			"Max retries exceeded",
		)
	}
	r.ExportCertificateSecret.Status.RetryCount++
	retryCount := r.ExportCertificateSecret.Status.RetryCount

	if err := r.Status().Update(ctx, r.ExportCertificateSecret); err != nil {
		r.Logger.Error(err, "Failed to update ExportCertificateSecret status")
		return r.eventHandler.ReturnEvent(
			r.Ctx,
			r.ExportCertificateSecret,
			r.ExportCertificateSecret.Status.RetryCount,
			err,
			"Failed to update ExportCertificateSecret status",
		)
	}
	// Add finalizer if not already present
	if r.ExportCertificateSecret.DeletionTimestamp.IsZero() {
		if err := r.addFinalizer(); err != nil {
			r.Logger.Error(err, "Failed to add finalizer")
			return r.eventHandler.ReturnEvent(
				r.Ctx,
				r.ExportCertificateSecret,
				retryCount,
				err,
				"Failed to add finalizer",
			)
		}
	}

	// Set the Kubernetes client
	r.KubeClient, err = utils.SetKubeClient()
	if err != nil {
		return r.eventHandler.ReturnEvent(
			r.Ctx,
			r.ExportCertificateSecret,
			retryCount,
			err,
			"Failed to set Kubernetes client",
		)
	}
	// Lookup the ServiceAccount referenced in the ExportCertificateSecret
	// This is used to authenticate with Key Vault using Workload Identity
	var serviceAccount corev1.ServiceAccount
	if err := r.Get(ctx, types.NamespacedName{
		Name:      r.ExportCertificateSecret.Spec.AzureKV.ServiceAccountRef.Name,
		Namespace: req.Namespace,
	}, &serviceAccount); err != nil {
		r.Logger.Error(err, "unable to fetch ServiceAccount")
		return r.eventHandler.ReturnEvent(
			r.Ctx,
			r.ExportCertificateSecret,
			retryCount,
			err,
			"Failed to fetch ServiceAccount",
		)
	}

	// Create an authorizer MSAL token for the service account based on annotations
	azureauth := &azureauth.WorkloadID{
		Ctx:            r.Ctx,
		Logger:         r.Logger,
		KubeClient:     r.KubeClient,
		ServiceAccount: serviceAccount,
		SaAudiences:    r.ExportCertificateSecret.Spec.AzureKV.ServiceAccountRef.Audiences,
		TokenProvider:  azureauth.NewTokenProvider,
	}
	err = azureauth.AuthorizerForWorkloadIdentity()
	if err != nil {
		r.Logger.Error(err, "failed to create authorizer")
		r.ExportCertificateSecret.Status.RetryCount++
		return r.eventHandler.ReturnEvent(
			r.Ctx,
			r.ExportCertificateSecret,
			retryCount,
			err,
			"Failed to create authorizer for Workload Identity",
		)
	}
	// Set the Azure Key Vault client
	// This client is used to interact with Azure Key Vault
	r.AkvClient = &akv.KeyVault{
		Ctx:      r.Ctx,
		Logger:   r.Logger,
		Client:   akv.NewAzureKeyVaultClient(azureauth.Authorizer),
		VaultUrl: r.ExportCertificateSecret.Spec.AzureKV.VaultUrl,
	}

	// Start secret sync process
	if err := r.syncSecrets(req.Namespace); err != nil {
		return r.eventHandler.ReturnEvent(
			r.Ctx,
			r.ExportCertificateSecret,
			retryCount,
			err,
			"Failed to sync secrets",
		)
	}
	// On successful sync, reset the retry count
	// This is to ensure that the next reconciliation does not trigger a retry
	r.ExportCertificateSecret.SetRetryCount(0)
	return r.eventHandler.ReturnEvent(
		r.Ctx,
		r.ExportCertificateSecret,
		r.ExportCertificateSecret.Status.RetryCount,
		nil,
		"",
	)
}

// Func preChecks performs pre-checks before the reconciliation loop starts
func (r *ExportCertificateSecretReconciler) preChecks(req ctrl.Request) error {
	// Check if the CRD is being deleted
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := r.Get(r.Ctx, types.NamespacedName{Name: vars.EXPORT_CERT_SECRET_CRD}, crd); err != nil {
		if errors.IsNotFound(err) {
			r.Logger.Info("CRD not found, skipping reconciliation")
			return nil
		}
		return err
	}
	// Lookup the ExportCertificateSecret object and assign it to the reconciler
	r.ExportCertificateSecret = &certdistributionv1alpha1.ExportCertificateSecret{}
	if err := r.Get(r.Ctx, req.NamespacedName, r.ExportCertificateSecret); err != nil {
		return err
	}

	return nil
}

// SyncSecret processes all key value pairs in the K8s Secret referenced in the ExportCertificateSecret
// and pushes them to Azure Key Vault based on the type of secret (crt or pem)
// It also checks if the secret already exists in Key Vault and compares the byte values
func (r *ExportCertificateSecretReconciler) SyncSecret(namespace string, certRef []certdistributionv1alpha1.CertificateSecretRef) error {
	var k8sSecret corev1.Secret
	secretAttributes := keyvault.SecretAttributes{}

	if r.OpenSSLRunners.Certificates.CertificateDetails.ExpirationTime != 0 {
		secretAttributes.Expires = func(t date.UnixTime) *date.UnixTime { return &t }(date.NewUnixTimeFromSeconds(r.OpenSSLRunners.Certificates.CertificateDetails.ExpirationTime))
	} else {
		r.Logger.Info("Certificate expiration time not found, this is probably due to lack of .crt secret being synced with AKV. See logs for more details")
	}

	for _, cert := range certRef {
		namespacedSecret := types.NamespacedName{
			Namespace: namespace,
			Name:      cert.SecretName,
		}
		if err := r.Get(r.Ctx, namespacedSecret, &k8sSecret); err != nil {
			r.Logger.Error(err, "unable to fetch referenced Kubernetes secret")
			return err
		}
		secretName := cert.KVSecretName
		// Check if the secret version exists in Key Vault based on byte comparison
		secretData, ok := k8sSecret.Data[cert.SecretKey]
		if !ok {
			r.Logger.Error(nil, "Secret key not found in Kubernetes secret")
			return fmt.Errorf("secret key not found in Kubernetes secret")
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
			case vars.BASE64EN_TEMPLATING:
				r.AkvClient.Secret.Value = base64.StdEncoding.EncodeToString([]byte(r.AkvClient.Secret.Value))
			}
			if r.AkvClient.Secret.SecretBundle.Value != nil {
				// Check if the secret version exists in Key Vault based on byte comparison
				ok, _ := utils.ByteCompare(r.Ctx, []byte(r.AkvClient.Secret.Value), *r.AkvClient.Secret.SecretBundle.Value)
				if ok {
					// Continue to next iteration since the current secret matches kv
					r.Logger.Info(fmt.Sprintf("Secret key matches that of in Key Vault %s", cert.SecretKey))
					continue
				}
			}
			// Push to Key Vault
			err := r.AkvClient.PushSecretToKeyVault()
			if err != nil {
				r.Logger.Error(err, fmt.Sprintf("unable to push secret to Key Vault %s", secretName))
				return err
			}
			r.Logger.Info(fmt.Sprintf("Successfully pushed secret to Key Vault %s", secretName))
			continue
		}
		if cert.Type == "pem" {
			switch cert.Templating {
			case vars.PEMTOPFX_TEMPLATING:
				r.AkvClient.Secret.Value = base64.StdEncoding.EncodeToString(r.OpenSSLRunners.Certificates.ConvertedFormats.PFXCertificate)
			case vars.BASE64EN_TEMPLATING:
				r.AkvClient.Secret.Value = base64.StdEncoding.EncodeToString(r.OpenSSLRunners.Certificates.PemBundle)
			}
			if r.AkvClient.Secret.SecretBundle.Value != nil {
				// Check if the secret version exists in Key Vault based on byte comparison
				ok, _ := utils.ByteCompare(r.Ctx, []byte(r.AkvClient.Secret.Value), *r.AkvClient.Secret.SecretBundle.Value)
				if ok {
					// Continue to next iteration since the current secret matches kv
					r.Logger.Info(fmt.Sprintf("Secret key matches that of in Key Vault %s", cert.SecretKey))
					continue
				}
			}
			// Push to Key Vault
			err := r.AkvClient.PushCertificateToKeyVault()
			if err != nil {
				r.Logger.Error(err, "unable to push pem to Key Vault")
				return err
			}
		}
	}

	return nil
}

// OpenSSLGenerator processes the certificates in the referenced Kubernetes secrets
// and generates the necessary OpenSSL runners for certificate processing
// It populates the OpenSSLRunners with the certificate data from the Kubernetes secrets
// OpenSSLRunner will generate expiration date and required cert formats defined in the ExportCertificateSecret
func (r *ExportCertificateSecretReconciler) OpenSSLGenerator(namespace string, certRefs []certdistributionv1alpha1.CertificateSecretRef) error {
	r.OpenSSLRunners = openssl.NewOpenSSLRunners(r.Ctx, r.Logger)

	for _, cert := range certRefs {
		namespacedSecret := types.NamespacedName{
			Namespace: namespace,
			Name:      cert.SecretName,
		}
		var k8sSecret corev1.Secret
		if err := r.Get(r.Ctx, namespacedSecret, &k8sSecret); err != nil {
			r.Logger.Error(err, "unable to fetch referenced Kubernetes secret")
			return err
		}
		// Check if the secret version exists in Key Vault based on byte comparison
		secretData, ok := k8sSecret.Data[cert.SecretKey]
		if !ok {
			r.Logger.Error(nil, "Secret key not found in Kubernetes secret")
			return fmt.Errorf("secret key not found in Kubernetes secret")
		}
		// Based on key in k8s secret we populate the OpenSSLRunners with the certificate data
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
			r.Logger.Error(fmt.Errorf("secret key with value not supported %s", cert.SecretKey), "")
			continue
		}
	}
	// Call the OpenSSL runners to process the certificates into proper formats
	if err := r.OpenSSLRunners.ProcessCertificates(); err != nil {
		return err
	}

	return nil
}

// DeleteSecret deletes the secrets from Key Vault sequentially
func (r *ExportCertificateSecretReconciler) DeleteSecret(certRef []certdistributionv1alpha1.CertificateSecretRef) error {
	if certRef == nil {
		certRef = r.ExportCertificateSecret.Spec.AzureKV.CertificateSecretRef
	}

	var errList []error
	// loop through the certificate references and delete them from Key Vault
	// based on the type of secret (crt or pem)
	for _, cert := range certRef {
		secretName := cert.KVSecretName
		r.AkvClient.Secret = &akv.Secret{
			Name: secretName,
		}
		// We delete the secret/cert then try to purge it if the OnDeletePurge flag is set
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

// syncSecrets syncs the secrets in Key Vault with the current state of the ExportCertificateSecret
func (r *ExportCertificateSecretReconciler) syncSecrets(namespace string) error {
	secretRefs := r.ExportCertificateSecret.Spec.AzureKV.CertificateSecretRef

	// Seed certificates
	if err := r.OpenSSLGenerator(namespace, secretRefs); err != nil {
		return err
	}
	if err := r.SyncSecret(namespace, secretRefs); err != nil {
		return err
	}

	return nil
}

// addFinalizer adds the finalizer to the ExportCertificateSecret object
func (r *ExportCertificateSecretReconciler) addFinalizer() error {
	if !utils.ContainsString(r.ExportCertificateSecret.Finalizers, vars.FINALIZER) {
		r.ExportCertificateSecret.Finalizers = append(r.ExportCertificateSecret.Finalizers, vars.FINALIZER)
		if err := r.Update(r.Ctx, r.ExportCertificateSecret); err != nil {
			return err
		}
	}
	return nil
}
