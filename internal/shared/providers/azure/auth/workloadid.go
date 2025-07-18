package workloadidentity

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // Alias the import

	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/go-logr/logr"
	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	AzureDefaultAudience = "api://AzureADTokenExchange"
)

type WorkloadID struct {
	Ctx            context.Context
	Logger         logr.Logger
	KubeClient     kubernetes.Interface
	ServiceAccount corev1.ServiceAccount
	SaAudiences    []string
	TokenProvider  tokenProviderFunc
	Authorizer     autorest.Authorizer
}

// GetKeyVaultClient creates a Key Vault client using the provided ServiceAccount and Vault URL
// AuthorizerForWorkloadIdentity returns an Azure authorizer for a workload identity.
// It takes the following parameters:
// - ctx: The context.Context for the operation.
// - kubeClient: The kubernetes.Clientset for accessing the Kubernetes API.
// - serviceAccount: The corev1.ServiceAccount representing the service account.
// - saAudience: A slice of strings representing the audiences for the service account.
// - tokenProvider: A function that provides the token for authentication.
//
// It returns an autorest.Authorizer and an error. The authorizer can be used to authenticate
// requests to Azure services using the workload identity. The error will be non-nil if there
// was an error retrieving the token or creating the authorizer.
//
// This function retrieves the client ID and tenant ID from the service account annotations.
// It then uses the kubeClient to fetch a service account token. Finally, it calls the tokenProvider
// function to create a token provider and returns a new BearerAuthorizer using the token provider.
func (aza *WorkloadID) AuthorizerForWorkloadIdentity() error {
	aadEndpoint := azure.PublicCloud.ActiveDirectoryEndpoint
	kvResource := azure.PublicCloud.KeyVaultEndpoint
	namespace := aza.ServiceAccount.Namespace
	saName := aza.ServiceAccount.Name

	clientID, ok := aza.ServiceAccount.Annotations["azure.workload.identity/client-id"]
	if !ok {
		aza.Logger.Error(fmt.Errorf("client ID annotation not found"), "ServiceAccount does not have the required client ID annotation")
		return fmt.Errorf("client ID annotation not found")
	}

	tenantID, ok := aza.ServiceAccount.Annotations["azure.workload.identity/tenant-id"]
	if !ok {
		aza.Logger.Error(fmt.Errorf("tenant ID annotation not found"), "ServiceAccount does not have the required tenant ID annotation")
		return fmt.Errorf("tenant ID annotation not found")
	}

	audiences := []string{AzureDefaultAudience}
	if len(aza.SaAudiences) > 0 {
		audiences = append(audiences, aza.SaAudiences...)
	}
	token, err := aza.FetchSAToken(namespace, saName, audiences)
	if err != nil {
		return err
	}
	tp, err := aza.TokenProvider(aza.Ctx, token, clientID, tenantID, aadEndpoint, kvResource)
	if err != nil {
		return err
	}
	aza.Authorizer = autorest.NewBearerAuthorizer(tp)
	return nil
}

func (aza *WorkloadID) FetchSAToken(namespace, saName string, audiences []string) (string, error) {
	kubeClient := aza.KubeClient.CoreV1()
	token, err := kubeClient.ServiceAccounts(namespace).CreateToken(aza.Ctx, saName, &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences: audiences,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	return token.Status.Token, nil
}

// tokenProvider satisfies the adal.OAuthTokenProvider interface.
type tokenProvider struct {
	accessToken string
}

type tokenProviderFunc func(ctx context.Context, token, clientID, tenantID, aadEndpoint, kvResource string) (adal.OAuthTokenProvider, error)

func NewTokenProvider(ctx context.Context, token, clientID, tenantID, aadEndpoint, kvResource string) (adal.OAuthTokenProvider, error) {
	// exchange token with Azure AccessToken
	cred := confidential.NewCredFromAssertionCallback(func(ctx context.Context, aro confidential.AssertionRequestOptions) (string, error) {
		return token, nil
	})
	cClient, err := confidential.New(fmt.Sprintf("%s%s/oauth2/token", aadEndpoint, tenantID), clientID, cred)
	if err != nil {
		return nil, err
	}
	scope := kvResource
	// .default needs to be added to the scope
	if !strings.Contains(kvResource, ".default") {
		scope = fmt.Sprintf("%s/.default", kvResource)
	}
	authRes, err := cClient.AcquireTokenByCredential(ctx, []string{
		scope,
	})
	if err != nil {
		return nil, err
	}
	return &tokenProvider{
		accessToken: authRes.AccessToken,
	}, nil
}

func (t *tokenProvider) OAuthToken() string {
	return t.accessToken
}
