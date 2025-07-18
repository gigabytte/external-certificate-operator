package keyvault

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/go-logr/logr"
)

// AzureKeyVaultClient is a wrapper around keyvault.BaseClient that implements KeyVaultClient
type AzureKeyVaultClient struct {
	keyvault.BaseClient
}

type KeyVaultClient interface {
	SetSecret(ctx context.Context, vaultBaseURL, secretName string, parameters keyvault.SecretSetParameters) (result keyvault.SecretBundle, err error)
	ImportCertificate(ctx context.Context, vaultBaseURL, certificateName string, parameters keyvault.CertificateImportParameters) (result keyvault.CertificateBundle, err error)
	DeleteSecret(ctx context.Context, vaultBaseURL, secretName string) (result keyvault.DeletedSecretBundle, err error)
	PurgeDeletedSecret(ctx context.Context, vaultBaseURL, secretName string) (result autorest.Response, err error)
	PurgeDeletedCertificate(ctx context.Context, vaultBaseURL, certificateName string) (result autorest.Response, err error)
	DeleteCertificate(ctx context.Context, vaultBaseURL, certificateName string) (result keyvault.DeletedCertificateBundle, err error)
	GetSecret(ctx context.Context, vaultBaseURL, secretName, version string) (result keyvault.SecretBundle, err error)
}

type KeyVault struct {
	Ctx      context.Context
	Logger   logr.Logger
	Client   KeyVaultClient
	VaultUrl string
	Secret   *Secret
}

type Secret struct {
	Name         string
	Value        string
	Attributes   keyvault.SecretAttributes
	SecretBundle keyvault.SecretBundle
}
