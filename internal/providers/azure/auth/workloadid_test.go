package workloadidentity

import (
	"context"
	"fmt"
	"testing"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/stretchr/testify/assert"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestAuthorizerForWorkloadIdentity(t *testing.T) {
	tests := []struct {
		name           string
		serviceAccount corev1.ServiceAccount
		saAudiences    []string
		tokenProvider  tokenProviderFunc
		expectedError  error
	}{
		{
			name: "Valid ServiceAccount",
			serviceAccount: corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-sa",
					Annotations: map[string]string{
						"azure.workload.identity/client-id": "test-client-id",
						"azure.workload.identity/tenant-id": "test-tenant-id",
					},
				},
			},
			saAudiences: []string{"test-audience"},
			tokenProvider: func(ctx context.Context, token, clientID, tenantID, aadEndpoint, kvResource string) (adal.OAuthTokenProvider, error) {
				return &tokenProvider{accessToken: "test-token"}, nil
			},
			expectedError: nil,
		},
		{
			name: "Missing Client ID Annotation",
			serviceAccount: corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-sa",
					Annotations: map[string]string{
						"azure.workload.identity/tenant-id": "test-tenant-id",
					},
				},
			},
			saAudiences:   []string{"test-audience"},
			tokenProvider: nil,
			expectedError: fmt.Errorf("client ID annotation not found"),
		},
		{
			name: "Missing Tenant ID Annotation",
			serviceAccount: corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-sa",
					Annotations: map[string]string{
						"azure.workload.identity/client-id": "test-client-id",
					},
				},
			},
			saAudiences:   []string{"test-audience"},
			tokenProvider: nil,
			expectedError: fmt.Errorf("tenant ID annotation not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset(&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: tt.serviceAccount.Namespace,
					Name:      tt.serviceAccount.Name,
				},
			})

			// Mock the service account token creation
			kubeClient.PrependReactor("create", "serviceaccounts/token", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				createAction := action.(k8stesting.CreateAction)
				tokenRequest := createAction.GetObject().(*authv1.TokenRequest)
				if len(tokenRequest.Spec.Audiences) > 0 && tokenRequest.Spec.Audiences[1] == "test-audience" && tt.serviceAccount.Name == "test-sa" {
					return true, &authv1.TokenRequest{
						Status: authv1.TokenRequestStatus{
							Token: "test-token",
						},
					}, nil
				}
				return true, nil, fmt.Errorf("failed to fetch service account token")
			})

			wi := &WorkloadID{
				Ctx:            context.TODO(),
				KubeClient:     kubeClient,
				ServiceAccount: tt.serviceAccount,
				SaAudiences:    tt.saAudiences,
				TokenProvider:  tt.tokenProvider,
			}

			fmt.Printf("Running test: %s\n", tt.name)
			err := wi.AuthorizerForWorkloadIdentity()
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, wi.Authorizer)
			}
		})
	}
}

func TestFetchSAToken(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		saName        string
		audiences     []string
		expectedToken string
		expectedError error
	}{
		{
			name:          "Valid Token",
			namespace:     "default",
			saName:        "test-sa",
			audiences:     []string{"test-audience"},
			expectedToken: "test-token",
			expectedError: nil,
		},
		{
			name:          "Invalid ServiceAccount",
			namespace:     "default",
			saName:        "invalid-sa",
			audiences:     []string{"test-audience"},
			expectedToken: "",
			expectedError: fmt.Errorf("failed to fetch service account token"),
		},
		{
			name:          "Invalid Audience",
			namespace:     "default",
			saName:        "test-sa",
			audiences:     []string{"invalid-audience"},
			expectedToken: "",
			expectedError: fmt.Errorf("failed to fetch service account token"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset(&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: tt.namespace,
					Name:      tt.saName,
				},
			})

			// Mock the service account token creation
			kubeClient.PrependReactor("create", "serviceaccounts/token", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				createAction := action.(k8stesting.CreateAction)
				tokenRequest := createAction.GetObject().(*authv1.TokenRequest)
				if tokenRequest.Spec.Audiences[0] == "test-audience" && tt.saName == "test-sa" {
					return true, &authv1.TokenRequest{
						Status: authv1.TokenRequestStatus{
							Token: "test-token",
						},
					}, nil
				}
				return true, nil, fmt.Errorf("failed to fetch service account token")
			})

			w := &WorkloadID{
				Ctx:        context.TODO(),
				KubeClient: kubeClient,
			}

			token, err := w.FetchSAToken(tt.namespace, tt.saName, tt.audiences)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}
