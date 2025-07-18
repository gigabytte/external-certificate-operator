package keyvault

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPushSecretToKeyVault(t *testing.T) {
	mockClient := &MockKeyVaultClient{}

	// Create a test response bundle with StatusCode
	resp := autorest.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}

	bundle := keyvault.SecretBundle{
		Value:    new(string),
		ID:       new(string),
		Response: resp,
	}
	*bundle.Value = "secret-value"

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

	// Mock the SetSecret call with proper response
	mockClient.On("SetSecret",
		mock.Anything,
		"https://example.vault.azure.net",
		"test-secret",
		mock.MatchedBy(func(params keyvault.SecretSetParameters) bool {
			return *params.Value == "secret-value"
		}),
	).Return(bundle, nil)

	err := akv.PushSecretToKeyVault()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestPushCertificateToKeyVault(t *testing.T) {
	mockClient := new(MockKeyVaultClient)

	// Create test response
	resp := autorest.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}

	certBundle := keyvault.CertificateBundle{
		Response: resp,
	}

	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name:  "test-cert",
			Value: "cert-value",
		},
	}

	mockClient.On("ImportCertificate",
		mock.Anything,
		"https://example.vault.azure.net",
		"test-cert",
		mock.MatchedBy(func(params keyvault.CertificateImportParameters) bool {
			return *params.Base64EncodedCertificate == "cert-value"
		}),
	).Return(certBundle, nil)

	err := akv.PushCertificateToKeyVault()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDeleteSecretFromKeyVault(t *testing.T) {
	mockClient := new(MockKeyVaultClient)

	// Create test response
	resp := autorest.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}

	deletedBundle := keyvault.DeletedSecretBundle{
		Response: resp,
	}

	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name: "test-secret",
		},
	}

	mockClient.On("DeleteSecret", mock.Anything, "https://example.vault.azure.net", "test-secret").Return(deletedBundle, nil)

	err := akv.DeleteSecretFromKeyVault()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestPurgeDeletedSecretFromKeyVault(t *testing.T) {
	mockClient := new(MockKeyVaultClient)

	// Create test response
	resp := autorest.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}

	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name: "test-secret",
		},
	}

	mockClient.On("PurgeDeletedSecret", mock.Anything, "https://example.vault.azure.net", "test-secret").Return(resp, nil)

	err := akv.PurgeDeletedSecretFromKeyVault()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestPurgeDeletedSecretFromKeyVaultWithPurgeProtection(t *testing.T) {
	mockClient := new(MockKeyVaultClient)

	// Create test response with 403 for purge protection
	resp := autorest.Response{
		Response: &http.Response{
			StatusCode: http.StatusForbidden,
		},
	}

	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name: "test-secret",
		},
	}

	mockClient.On("PurgeDeletedSecret", mock.Anything, "https://example.vault.azure.net", "test-secret").
		Return(resp, fmt.Errorf("Operation \"purge\" is not allowed because purge protection is enabled"))

	err := akv.PurgeDeletedSecretFromKeyVault()
	assert.NoError(t, err) // Should return nil when purge protection is enabled
	mockClient.AssertExpectations(t)
}

func TestPurgeDeletedCertificateFromKeyVault(t *testing.T) {
	mockClient := new(MockKeyVaultClient)

	// Create test response
	resp := autorest.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}

	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name: "test-cert",
		},
	}

	mockClient.On("PurgeDeletedCertificate", mock.Anything, "https://example.vault.azure.net", "test-cert").Return(resp, nil)

	err := akv.PurgeDeletedCertificateFromKeyVault()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDeleteCertificateFromKeyVault(t *testing.T) {
	mockClient := new(MockKeyVaultClient)

	// Create test response
	resp := autorest.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}

	deletedBundle := keyvault.DeletedCertificateBundle{
		Response: resp,
	}

	akv := &KeyVault{
		Ctx:      context.TODO(),
		Client:   mockClient,
		VaultUrl: "https://example.vault.azure.net",
		Secret: &Secret{
			Name: "test-cert",
		},
	}

	mockClient.On("DeleteCertificate", mock.Anything, "https://example.vault.azure.net", "test-cert").Return(deletedBundle, nil)

	err := akv.DeleteCertificateFromKeyVault()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}
