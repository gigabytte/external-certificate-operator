package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	certdistributionv1alpha1 "github.com/gigabytte/external-certificate-operator/api/v1alpha1"
	"github.com/gigabytte/external-certificate-operator/internal/log"
)

// RemoveFinalizersFromAllCustomResources removes finalizers from all instances of the specified custom resource type
func RemoveFinalizersFromAllCustomResources(ctx context.Context, c client.Client, resourceType interface{}) error {
	log := log.FromContext(ctx)

	// Determine the resource type and list
	var list client.ObjectList
	switch resourceType.(type) {
	case *certdistributionv1alpha1.ExportCertificateSecret:
		list = &certdistributionv1alpha1.ExportCertificateSecretList{}
	case *certdistributionv1alpha1.ImportCertificateSecret:
		list = &certdistributionv1alpha1.ImportCertificateSecretList{}
	default:
		return fmt.Errorf("unsupported resource type")
	}

	// List all instances of the specified resource type
	if err := c.List(ctx, list); err != nil {
		log.Error(err, "unable to list resources")
		return err
	}

	// Iterate over the items and remove the finalizer
	items := reflect.ValueOf(list).Elem().FieldByName("Items")
	for i := 0; i < items.Len(); i++ {
		item := items.Index(i).Addr().Interface().(metav1.Object)
		if ContainsString(item.GetFinalizers(), FINALIZER) {
			item.SetFinalizers(RemoveString(item.GetFinalizers(), FINALIZER))
			if err := c.Update(ctx, item.(client.Object)); err != nil {
				log.Error(err, "unable to remove finalizer", "name", item.GetName(), "namespace", item.GetNamespace())
				return err
			}
			log.Info("Removed finalizer", "name", item.GetName(), "namespace", item.GetNamespace())
		}
	}

	return nil
}

// SetKubeClient sets the Kubernetes client
func SetKubeClient() (kubernetes.Interface, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}

// IsBase64Encoded checks if a given byte slice is base64 encoded
func IsBase64Encoded(data []byte) bool {
	// Try to decode the byte slice
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return false
	}

	// Re-encode the decoded data
	reencoded := base64.StdEncoding.EncodeToString(decoded)

	// Check if the re-encoded string matches the original byte slice
	return string(data) == reencoded
}

// ByteCompare compares the byte slices of a Kubernetes secret and a Key Vault secret
func ByteCompare(ctx context.Context, k8sSecret []byte, kvSecret string) (bool, error) {
	// Try to decode the Key Vault secret from base64
	decodedKVSecret, err := base64.StdEncoding.DecodeString(kvSecret)
	if err != nil {
		// If decoding fails, assume kvSecret is not base64 encoded and compare directly with decoded k8s secret
		return bytes.Equal([]byte(kvSecret), k8sSecret), nil
	}
	// If decoding succeeds, compare the decoded value with the decoded Kubernetes secret
	return bytes.Equal(decodedKVSecret, k8sSecret), nil
}

// ContainsString checks if a string is present in a slice of strings.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// RemoveString removes a string from a slice of strings.
func RemoveString(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return result
}

// ConditionExists checks if a given condition already exists in a slice of conditions.
func ConditionExists(conditions []metav1.Condition, newCondition metav1.Condition) bool {
	for _, condition := range conditions {
		if condition.Type == newCondition.Type && condition.Status == newCondition.Status && condition.Reason == newCondition.Reason {
			return true
		}
	}
	return false
}

func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

// CreateOrPatchK8sSecret creates or patches a Kubernetes Secret.
func CreateOrPatchK8sSecret(ctx context.Context, kubeClient kubernetes.Interface, name string, data map[string][]byte, namespace string) error {
	// Create a new Secret object
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
		Type: corev1.SecretTypeTLS, // Set the secret type to kubernetes.io/tls
	}

	// Check if the Secret already exists
	existingSecret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Secret does not exist, create it
			_, err = kubeClient.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			return nil
		} else {
			// Other error occurred
			return err
		}
	}

	// Merge the new data with the existing data
	for key, value := range data {
		existingSecret.Data[key] = value
	}

	// Check if the data has changed
	if equal(existingSecret.Data, data) {
		// Data has not changed, no need to patch
		return nil
	}

	// Secret exists and data has changed, patch it
	oldData, err := json.Marshal(existingSecret)
	if err != nil {
		return err
	}

	// Update the existing secret with the new data
	existingSecret.Data = data

	newData, err := json.Marshal(existingSecret)
	if err != nil {
		return err
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, corev1.Secret{})
	if err != nil {
		return err
	}

	_, err = kubeClient.CoreV1().Secrets(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}

// equal compares two maps for equality
func equal(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if !bytes.Equal(v, b[k]) {
			return false
		}
	}
	return true
}
