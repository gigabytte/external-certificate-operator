package keyvault

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/stretchr/testify/mock"
)

type MockKeyVaultClient struct {
	mock.Mock
	Response *http.Response
}

func (m *MockKeyVaultClient) GetSecretSender(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if m.Response != nil {
		return m.Response, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

func (m *MockKeyVaultClient) GetSecretResponder(resp *http.Response) (result keyvault.SecretBundle, err error) {
	args := m.Called(resp)
	return args.Get(0).(keyvault.SecretBundle), args.Error(1)
}

func (m *MockKeyVaultClient) SetSecret(ctx context.Context, vaultBaseURL, secretName string, parameters keyvault.SecretSetParameters) (result keyvault.SecretBundle, err error) {
	args := m.Called(ctx, vaultBaseURL, secretName, parameters)
	return args.Get(0).(keyvault.SecretBundle), args.Error(1)
}

func (m *MockKeyVaultClient) ImportCertificate(ctx context.Context, vaultBaseURL, certificateName string, parameters keyvault.CertificateImportParameters) (result keyvault.CertificateBundle, err error) {
	args := m.Called(ctx, vaultBaseURL, certificateName, parameters)
	return args.Get(0).(keyvault.CertificateBundle), args.Error(1)
}

func (m *MockKeyVaultClient) DeleteSecret(ctx context.Context, vaultBaseURL, secretName string) (result keyvault.DeletedSecretBundle, err error) {
	args := m.Called(ctx, vaultBaseURL, secretName)
	return args.Get(0).(keyvault.DeletedSecretBundle), args.Error(1)
}

func (m *MockKeyVaultClient) PurgeDeletedSecret(ctx context.Context, vaultBaseURL, secretName string) (result autorest.Response, err error) {
	args := m.Called(ctx, vaultBaseURL, secretName)
	return args.Get(0).(autorest.Response), args.Error(1)
}

func (m *MockKeyVaultClient) PurgeDeletedCertificate(ctx context.Context, vaultBaseURL, certificateName string) (result autorest.Response, err error) {
	args := m.Called(ctx, vaultBaseURL, certificateName)
	return args.Get(0).(autorest.Response), args.Error(1)
}

func (m *MockKeyVaultClient) DeleteCertificate(ctx context.Context, vaultBaseURL, certificateName string) (result keyvault.DeletedCertificateBundle, err error) {
	args := m.Called(ctx, vaultBaseURL, certificateName)
	return args.Get(0).(keyvault.DeletedCertificateBundle), args.Error(1)
}

func (m *MockKeyVaultClient) GetSecret(ctx context.Context, vaultBaseURL, secretName, version string) (result keyvault.SecretBundle, err error) {
	args := m.Called(ctx, vaultBaseURL, secretName, version)
	return args.Get(0).(keyvault.SecretBundle), args.Error(1)
}

func BoolPtr(b bool) *bool {
	return &b
}
