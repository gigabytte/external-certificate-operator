package keyvault

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/Azure/go-autorest/autorest"
)

var (
	purgeInitialDelayMultiplier = 5 * time.Second
	purgeMaxDelay               = 30 * time.Second
	MaxRetries                  = 3
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

	secretName := strings.ToLower(akv.Secret.Name)
	params := keyvault.SecretSetParameters{Value: &akv.Secret.Value, SecretAttributes: &akv.Secret.Attributes}

	res, err := akv.Client.SetSecret(akv.Ctx, akv.VaultUrl, secretName, params)
	if res.StatusCode != 200 {
		akv.Logger.Info("unable to set secret in Key Vault", "error", err.Error(), "secretName", secretName)
		return err
	}
	akv.Secret.SecretBundle = res

	return nil
}

// PushCertificateToKeyVault pushes a certificate to Azure Key Vault
func (akv *KeyVault) PushCertificateToKeyVault() error {

	certName := strings.ToLower(akv.Secret.Name)
	params := keyvault.CertificateImportParameters{Base64EncodedCertificate: &akv.Secret.Value}

	res, err := akv.Client.ImportCertificate(akv.Ctx, akv.VaultUrl, certName, params)
	if res.StatusCode != 200 {
		akv.Logger.Info("unable to set certificate in Key Vault", "error", err.Error(), "certName", certName)
		return err
	}
	return nil
}

// DeleteSecretFromKeyVault deletes a secret to Azure Key Vault
func (akv *KeyVault) DeleteSecretFromKeyVault() error {
	akv.Logger.Info("Deleting secret from Key Vault")
	//
	secretName := strings.ToLower(akv.Secret.Name)

	_, err := akv.Client.DeleteSecret(akv.Ctx, akv.VaultUrl, secretName)
	akv.Logger.Info("Deleting secret from Key Vault")
	if err != nil {
		akv.Logger.Info("unable to delete secret from Key Vault", "secretName", secretName)
	}

	akv.Logger.Info("Successfully deleted secret from Key Vault", "secretName", secretName)
	return nil
}

// PurgeDeletedSecretFromKeyVault purges a deleted secret from Azure Key Vault
func (akv *KeyVault) PurgeDeletedSecretFromKeyVault() error {

	secretName := strings.ToLower(akv.Secret.Name)
	var err error

	for i := 0; i < MaxRetries; i++ {
		res, err := akv.Client.PurgeDeletedSecret(akv.Ctx, akv.VaultUrl, secretName)
		// Check if error is due to purge protection being enabled
		if res.StatusCode == 403 &&
			strings.Contains(err.Error(), "purge protection is enabled") {
			akv.Logger.Info("Skipping purge operation - purge protection is enabled",
				"secretName", secretName,
				"message", "Secret will be automatically purged after retention period")
			return nil
		}
		if res.StatusCode == 200 {
			akv.Logger.Info("Successfully purged secret from Key Vault", "secretName", secretName)
			return nil
		}

		akv.Logger.Info("Failed to purge secret from Key Vault, retrying...",
			"attempt", i+1,
			"error", err)
		time.Sleep(time.Duration(math.Min(float64(purgeMaxDelay),
			float64(purgeInitialDelayMultiplier)*math.Pow(2, float64(i)))))
	}
	return err
}

// PurgeDeletedCertificateFromKeyVault purges a deleted certificate from Azure Key Vault
func (akv *KeyVault) PurgeDeletedCertificateFromKeyVault() error {

	certName := strings.ToLower(akv.Secret.Name)
	var err error

	for i := 0; i < MaxRetries; i++ {
		res, err := akv.Client.PurgeDeletedCertificate(akv.Ctx, akv.VaultUrl, certName)
		// Check if error is due to purge protection being enabled
		if res.StatusCode == 403 &&
			strings.Contains(err.Error(), "purge protection is enabled") {
			akv.Logger.Info("Skipping purge operation - purge protection is enabled",
				"certName", certName,
				"message", "Certificate will be automatically purged after retention period")
			return nil
		}
		if res.StatusCode == 200 {
			akv.Logger.Info("Successfully purged certificate from Key Vault", "certName", certName)
			return nil
		}

		akv.Logger.Info("Failed to purge certificate from Key Vault, retrying...",
			"attempt", i+1,
			"error", err)
		time.Sleep(time.Duration(math.Min(float64(purgeMaxDelay),
			float64(purgeInitialDelayMultiplier)*math.Pow(2, float64(i)))))
	}
	return err
}

// DeleteCertificateFromKeyVault pushes a certificate to Azure Key Vault
func (akv *KeyVault) DeleteCertificateFromKeyVault() error {

	certName := strings.ToLower(akv.Secret.Name)

	res, err := akv.Client.DeleteCertificate(akv.Ctx, akv.VaultUrl, certName)
	if res.StatusCode != 200 {
		akv.Logger.Info("unable to delete certificate in Key Vault", "error", err.Error(), "certName", certName)
		return err
	}

	akv.Logger.Info("Successfully deleted certificate from Key Vault", "certName", certName)
	return nil
}

// checkSecretExistence checks if the specified secrets already exist in Azure Key Vault
func (akv *KeyVault) CheckKVSecretExistence() error {

	secretName := strings.ToLower(akv.Secret.Name)
	akv.Secret.SecretBundle = keyvault.SecretBundle{}

	// Check secret existence in Key Vault
	res, err := akv.Client.GetSecret(akv.Ctx, akv.VaultUrl, secretName, "")
	if res.StatusCode != 200 {
		// If the secret is not found (404), log the info and continue
		if detailedErr, ok := err.(autorest.DetailedError); ok && detailedErr.StatusCode == 404 {
			akv.Logger.Info("secret not found in Key Vault", "secretNameNotFound", secretName)
			return nil
		}
		// Log any other errors and return
		akv.Logger.Info("Unable to query Secrets from KeyVault", "error", err.Error(), "querySecretNameErr", secretName)
		return err
	}
	akv.Secret.SecretBundle = res

	return nil
}
