package keyvault

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPushSecretToKeyVault(t *testing.T) {
	mockClient := new(MockKeyVaultClient)
	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name:  "test-secret",
			Value: "secret-value",
			Attributes: keyvault.SecretAttributes{
				Enabled: BoolPtr(true),
			},
		},
	}

	mockClient.On("SetSecret", mock.Anything, "https://example.vault.azure.net", "test-secret", mock.Anything).Return(keyvault.SecretBundle{}, nil)

	err := akv.PushSecretToKeyVault()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestPushCertificateToKeyVault(t *testing.T) {
	mockClient := new(MockKeyVaultClient)
	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name:  "test-cert",
			Value: "cert-value",
		},
	}

	mockClient.On("ImportCertificate", mock.Anything, "https://example.vault.azure.net", "test-cert", mock.Anything).Return(keyvault.CertificateBundle{}, nil)

	err := akv.PushCertificateToKeyVault()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDeleteSecretFromKeyVault(t *testing.T) {
	mockClient := new(MockKeyVaultClient)
	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name: "test-secret",
		},
	}

	mockClient.On("DeleteSecret", mock.Anything, "https://example.vault.azure.net", "test-secret").Return(keyvault.DeletedSecretBundle{}, nil)

	err := akv.DeleteSecretFromKeyVault()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestPurgeDeletedSecretFromKeyVault(t *testing.T) {
	mockClient := new(MockKeyVaultClient)
	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name: "test-secret",
		},
	}

	mockClient.On("PurgeDeletedSecret", mock.Anything, "https://example.vault.azure.net", "test-secret").Return(autorest.Response{}, nil)

	err := akv.PurgeDeletedSecretFromKeyVault()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestPurgeDeletedCertificateFromKeyVault(t *testing.T) {
	mockClient := new(MockKeyVaultClient)
	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name: "test-cert",
		},
	}

	mockClient.On("PurgeDeletedCertificate", mock.Anything, "https://example.vault.azure.net", "test-cert").Return(autorest.Response{}, nil)

	err := akv.PurgeDeletedCertificateFromKeyVault()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDeleteCertificateFromKeyVault(t *testing.T) {
	mockClient := new(MockKeyVaultClient)
	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name: "test-cert",
		},
	}

	mockClient.On("DeleteCertificate", mock.Anything, "https://example.vault.azure.net", "test-cert").Return(keyvault.DeletedCertificateBundle{}, nil)

	err := akv.DeleteCertificateFromKeyVault()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestCheckKVSecretExistence(t *testing.T) {
	mockClient := new(MockKeyVaultClient)
	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name: "test-secret",
		},
	}

	mockClient.On("GetSecret", mock.Anything, "https://example.vault.azure.net", "test-secret", "").Return(keyvault.SecretBundle{}, nil)

	err := akv.CheckKVSecretExistence()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestCheckKVSecretExistenceNotFound(t *testing.T) {
	mockClient := new(MockKeyVaultClient)
	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name: "test-secret",
		},
	}

	mockClient.On("GetSecret", mock.Anything, "https://example.vault.azure.net", "test-secret", "").Return(keyvault.SecretBundle{}, autorest.DetailedError{StatusCode: 404})

	err := akv.CheckKVSecretExistence()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}
