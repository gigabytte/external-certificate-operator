# API Reference

## Controller Manager Flags

```bash
Usage of /manager:
  -allowed-import-cross-namespaces string
        Comma-separated list of allowed namespaces that a given ImportCertficateSecret object can create secret in outside of its native namespace (ie. Cross ns secret creation)
  -enable-http2
        If set, HTTP/2 will be enabled for the metrics and webhook servers
  -health-probe-bind-address string
        The address the probe endpoint binds to. (default ":8081")
  -kubeconfig string
        Paths to a kubeconfig. Only required if out-of-cluster.
  -leader-elect
        Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
  -metrics-bind-address string
        The address the metric endpoint binds to. (default ":8080")
  -metrics-secure
        If set the metrics endpoint is served securely
  -zap-devel
        Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error) (default true)
  -zap-encoder value
        Zap log encoding (one of 'json' or 'console')
  -zap-log-level value
        Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
  -zap-stacktrace-level value
        Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').
  -zap-time-encoding value
        Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'). Defaults to 'epoch'.
```

## Packages

- [external-certificate.io/v1alpha1](#externalcertificateiov1alpha1)

## external-certificate.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the cert-distribution v1alpha1 API group

### Resource Types

- [ExportCertificateSecret](#exportcertificatesecret)
- [ImportCertificateSecret](#importcertificatesecret)

#### CertificateSecretRef

CertificateSecretRef represents the reference to the certificate secret

_Appears in:_
- [ExportAzureKVProvider](#exportazurekvprovider)
- [ImportAzureKVProvider](#importazurekvprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `KVSecretName` _string_ | Name of secret to be pushed to AKV | none | none |
| `SecretKey` _string_ | Name of K8s Secret key to be pushed to Azure KV | none | none |
| `SecretName` _string_ |  Name of K8s Secret where associated exist to be pushed to Azure KV  | none | none |
| `Templating` _string_ | Possible templating function `base64encode or pemtopfx` performs operation agaist secret before being pushed to Azure KV | `pemtopfx` is default for keys notes with `tls-combined.pem` | `base64encode` can only be used with standard cert types (ie. .crt). `pemtopfx` can only be used with cert types `tls-combined.pem` |

#### ExportAzureKVProvider

AzureKVProvider represents the Azure Key Vault provider configuration

_Appears in:_
- [ExportCertificateSecretSpec](#exportcertificatesecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vaultUrl` _string_ | Url of Azure KeyVault instance where secrets will be stored |  |  |
| `serviceAccountRef` _[ServiceAccountRef](#serviceaccountref)_ | Workload ID annotated service account running the `ExportCertificateSecret` resource | None | None |
| `certificateSecretRef` _[CertificateSecretRef](#certificatesecretref)_ |  |  |  |
| `ScanInterval` _string_ | Interval to scan destination Azure KeyVault for changes | 5 minutes | None |
| `OnDeletePurge` _bool_ | Upon cleanup of `ExportCertificateSecret` secrets will be deleted from KeyVault with purge functionality | `true` | none |

#### ExportCertificateSecret

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `external-certificate.io/v1alpha1` | | |
| `kind` _string_ | `ExportCertificateSecret` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ExportCertificateSecretSpec](#exportcertificatesecretspec)_ |  |  |  |

#### ExportCertificateSecretSpec

ExportCertificateSecretSpec defines the desired state of ExportCertificateSecret

_Appears in:_
- [ExportCertificateSecret](#exportcertificatesecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `azurekv` _[ExportAzureKVProvider](#exportazurekvprovider)_ |  |  |  |

#### ImportAzureKVProvider

ImportAzureKVProvider represents the Azure Key Vault provider configuration

_Appears in:_
- [ImportCertificateSecretSpec](#importcertificatesecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vaultUrl` _string_ | Url of Azure KeyVault instance where secrets will be stored |  |  |
| `serviceAccountRef` _[ServiceAccountRef](#serviceaccountref)_ | Workload ID annotated service account running the `ImportCertificateSecret` resource |  |  |
| `CertificateSecretRef` _[CertificateSecretRef](#certificateSecretRef) array_ |  |  |  |
| `secretNamespace` _string | Destination namespace where secret should be created. Allowed list of namespaces for cross namespace secret creation is handled by operator manager web server flag `--allowed-import-cross-namespaces=""` | Left blank defaults to object namespace. |  |
| `scanInterval` _integer_ | Interval at which kubernetes should reconcile (refresh) and scan for new changes in minutes | `5` in minutes |  |

#### ImportCertificateSecret

ImportCertificateSecret is the Schema for the importcertificatesecrets API

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `external-certificate.io/v1alpha1` | | |
| `kind` _string_ | `ImportCertificateSecret` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ImportCertificateSecretSpec](#importcertificatesecretspec)_ |  |  |  |

#### ImportCertificateSecretSpec

ImportCertificateSecretSpec defines the desired state of ImportCertificateSecret

_Appears in:_
- [ImportCertificateSecret](#importcertificatesecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `azurekv` _[ImportAzureKVProvider](#importazurekvprovider)_ | AzureKV is the Azure Key Vault provider configuration<br />def in exportcertificatesecret_types.go |  |  |

#### ServiceAccountRef

ServiceAccountRef represents a reference to a Kubernetes service account

_Appears in:_
- [ExportAzureKVProvider](#exportazurekvprovider)
- [ImportAzureKVProvider](#importazurekvprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of service account whom owns the `ExportCertificateSecret` or `ImportCertificateSecret` in the same namespace |  |  |

## Docs Generator

https://github.com/elastic/crd-ref-docs

```bash
crd-ref-docs
    --source-path=api/v1alpha1/ \
    --renderer=markdown \
    --config=api/v1alpha1/docs/config.yaml \
    --output-path=api/v1alpha1/docs/ \
    --output-mode=group
```
