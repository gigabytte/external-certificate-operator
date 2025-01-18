package keyvault

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/gigabytte/external-certificate-operator/internal/log"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/Azure/go-autorest/autorest"
)

var (
	purgeInitialDelayMultiplier = 5 * time.Second
	purgeMaxDelay               = 30 * time.Second
	maxRetries                  = 5
)

// SetSecret implements the KeyVaultClient interface
func (c *AzureKeyVaultClient) SetSecret(ctx context.Context, vaultBaseURL, secretName string, parameters keyvault.SecretSetParameters) (result keyvault.SecretBundle, err error) {
	return c.BaseClient.SetSecret(ctx, vaultBaseURL, secretName, parameters)
}

// NewAzureKeyVaultClient creates a new AzureKeyVaultClient with the given authorizer
func NewAzureKeyVaultClient(authorizer autorest.Authorizer) *AzureKeyVaultClient {
	client := keyvault.New()
	client.Authorizer = authorizer
	return &AzureKeyVaultClient{
		BaseClient: client,
	}
}

// PushSecretToKeyVault pushes a secret to Azure Key Vault
func (akv *KeyVault) PushSecretToKeyVault() error {
	log := log.FromContext(akv.Ctx)
	secretName := strings.ToLower(akv.Secret.Name)
	params := keyvault.SecretSetParameters{Value: &akv.Secret.Value, SecretAttributes: &akv.Secret.Attributes}

	secBundle, err := akv.Client.SetSecret(akv.Ctx, akv.VaultUrl, secretName, params)
	if err != nil {
		log.Error(err, "unable to set secret in Key Vault", "secretName", secretName)
		return err
	}
	akv.Secret.SecretBundle = secBundle

	return nil
}

// PushCertificateToKeyVault pushes a certificate to Azure Key Vault
func (akv *KeyVault) PushCertificateToKeyVault() error {
	log := log.FromContext(akv.Ctx)
	certName := strings.ToLower(akv.Secret.Name)
	params := keyvault.CertificateImportParameters{Base64EncodedCertificate: &akv.Secret.Value}

	_, err := akv.Client.ImportCertificate(akv.Ctx, akv.VaultUrl, certName, params)
	if err != nil {
		log.Error(err, "unable to set certificate in Key Vault", "certName", certName)
		return err
	}
	return nil
}

// DeleteSecretFromKeyVault deletes a secret to Azure Key Vault
func (akv *KeyVault) DeleteSecretFromKeyVault() error {
	log := log.FromContext(akv.Ctx)
	secretName := strings.ToLower(akv.Secret.Name)

	_, err := akv.Client.DeleteSecret(akv.Ctx, akv.VaultUrl, secretName)
	if err != nil {
		log.Info("unable to delete secret from Key Vault", "secretName", secretName)
	}

	log.Info("Successfully deleted secret from Key Vault", "secretName", secretName)
	return nil
}

// PurgeDeletedSecretFromKeyVault purges a deleted secret from Azure Key Vault
func (akv *KeyVault) PurgeDeletedSecretFromKeyVault() error {
	log := log.FromContext(akv.Ctx)
	secretName := strings.ToLower(akv.Secret.Name)
	var err error

	for i := 0; i < maxRetries; i++ {
		_, err := akv.Client.PurgeDeletedSecret(akv.Ctx, akv.VaultUrl, secretName)
		if err == nil {
			log.Info("Successfully purged secret from Key Vault", "secretName", secretName)
			return nil
		}
		log.Info("Failed to purge secret from Key Vault, retrying...", "attempt", i+1, "error", err)
		time.Sleep(time.Duration(math.Min(float64(purgeMaxDelay), float64(purgeInitialDelayMultiplier)*math.Pow(2, float64(i)))))
	}
	return err
}

// PurgeDeletedCertificateFromKeyVault purges a deleted certificate from Azure Key Vault
func (akv *KeyVault) PurgeDeletedCertificateFromKeyVault() error {
	log := log.FromContext(akv.Ctx)
	certName := strings.ToLower(akv.Secret.Name)
	var err error

	for i := 0; i < maxRetries; i++ {
		_, err := akv.Client.PurgeDeletedCertificate(akv.Ctx, akv.VaultUrl, certName)
		if err == nil {
			log.Info("Successfully purged certificate from Key Vault", "certName", certName)
			return nil
		}
		log.Info("Failed to purge certificate from Key Vault, retrying...", "attempt", i+1, "error", err)
		time.Sleep(time.Duration(math.Min(float64(purgeMaxDelay), float64(purgeInitialDelayMultiplier)*math.Pow(2, float64(i)))))
	}
	return err
}

// DeleteCertificateFromKeyVault pushes a certificate to Azure Key Vault
func (akv *KeyVault) DeleteCertificateFromKeyVault() error {
	log := log.FromContext(akv.Ctx)
	certName := strings.ToLower(akv.Secret.Name)

	_, err := akv.Client.DeleteCertificate(akv.Ctx, akv.VaultUrl, certName)
	if err != nil {
		log.Error(err, "unable to delete certificate in Key Vault", "certName", certName)
		return err
	}

	log.Info("Successfully deleted certificate from Key Vault", "certName", certName)
	return nil
}

// checkSecretExistence checks if the specified secrets already exist in Azure Key Vault
func (akv *KeyVault) CheckKVSecretExistence() error {
	log := log.FromContext(akv.Ctx)
	secretName := strings.ToLower(akv.Secret.Name)
	akv.Secret.SecretBundle = keyvault.SecretBundle{}

	// Check secret existence in Key Vault
	res, err := akv.Client.GetSecret(akv.Ctx, akv.VaultUrl, secretName, "")
	if err != nil {
		// If the secret is not found (404), log the info and continue
		if detailedErr, ok := err.(autorest.DetailedError); ok && detailedErr.StatusCode == 404 {
			log.Info("secret not found in Key Vault", "secretNameNotFound", secretName)
			return nil
		}
		// Log any other errors and return
		log.Error(err, "Unable to query Secrets from KeyVault", "querySecretNameErr", secretName)
		return err
	}
	akv.Secret.SecretBundle = res

	return nil
}
