# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [1.2.8] - 2025-07-09

### Bugfix

- Fixes issues with retry counter not being incremented properly
- Fixes issues with cert expiration generation with bad error checking causing false positive failure to occur.

## [1.2.7] - 2025-06-27

### Bugfix

- Updated logic when purging secrets/certs from AKV, controller will no longer hang when purge protection is enabled upon deleting manifests during finalization
- Removed secret compare logic to prevent false positives during k8s secret updates. When k8s secret is updated a new secret version is created in keyvault

## [1.2.6] - 2025-06-13

### Bugfix

- Namespace events created by `handleSuccess` method in the event_handler.go file had wrong `reason` value reported. Changed from `Success` to `CertificateProcessingSucceeded` to support existing dashboard / monitoring solutions

## [1.2.5] - 2025-06-13

### Bugfix

- Fixes issues when the contents of a given secret are updated (ie. labels, annotations, data) doesn't trigger reconciliation in the `ExportCertificateSecret` controller
- Refactors code for the `ExportCertificateSecret` controller to simplify and improve readability

### Changed

- `scanInterval` attribute is no longer available/ required for `ExportCertificateSecret`, the controller will retry to a maximum amount of 5 times at a max length of 30 minutes of requeue if failure occurs. The is outside of the normal reconcile process which is triggered via k8s secret `patch/create` and `ExportCertificateSecret` `patch/create/delete`
- Upgrades to Go version 1.24.4 and fixes CVEs found

## [1.2.4] - 2025-05-09

### Bugfix

- Adds the ability for users to define more options in operator helm chart for attributes in webhook certificate
- Updates base image to use latest available alpine images
- Removed helmchart app version to VERSION file validation due to being redundant
- Fixes openssl tests, new dynamic cert generator helper introduced for testing
- updated some pkgs to remediate high vulnrs

## [1.2.3] - 2025-03-10

### Bugfix

- Fixes PDB label selector to reflect proper labels on pods

## [1.2.2] - 2025-01-25

### Changed

- Improves logging format to pure json format for better log ingestion
- Updates pipeline to point to new Artifactory repos
- Upgrade vulnr packages to latest versions detected by GhAzDO

## [1.2.1] - 2025-01-17

### Changed

- Fixed issues with caBundle def when defining ca Bundle for webhook services. Defining attributes for caBundle now easier see helm chart values for details.

## [1.2.0] - 2024-12-18

### Added

- **BREAKING** `caBundleSecret` value reworked in Helmchart values to support ca bundle being passed as secret or inline value. See Values.yaml in operator helmchart for defaults
- Secrets created in AKV via the `ExportCertificateSecret` manifest now have expiration date set to that of certificate expiration date
- kube-rbac-proxy now supports mounting tls cert for secure comms in turn removing deprecation warnings
- CRD deleting no longer forces finalizers to trigger delaying deleting of CRD on stubborn keyvault scenarios
  - RBAC roles updated in operator helm chart to reflect CRD event watching
- Code refactored to reduce complexities
- Upgraded to golang 1.22.x
- Pipeline now uses golang taskfile template
- Events are now created in namespace where API request is sourced adding more visibility for consumers of operator

### Bugfix

- Validation added to enforce name of `CertificateSecretRef.SecretName` to be the same for all secret keys referenced for `ExportCertificateSecret`
- State is now tracked for all individual secret keys being exported to key vault. Changes to secret key references will be updated in AKV
  - Operator CRD updated with new attribute too sync state

## [1.1.1] - 2024-10-16

### Bugfix

- Adds MSAL auth errors to retry logic on reconcile
- Fixes syntax bug with crd helmchart

## [1.1.0] - 2024-10-11

### Added

- Adds the ability to define a destination namespace where the secret should be created for the `ImportCertificateSecret` by the name of `Spec.AzureKV.SecretNamespace`.
- Admin can control allowed values for `SecretNamespace` via a comma separated list of allowed namespaces at the deployment spec for the operator via an arg flag `--allowed-import-cross-namespaces=foo,bar`.

### Bugfix

- Fixes validation and defaulting methods for ImportCertificateSecret specs to enforce some constraints on yaml spec.
- Fixes issues with manually deleted k8s secrets not being recreated by automatic reconcile loop from cluster events. When manually deleting the secret created by the `ImportCertificateSecret` such event will annotate the `ImportCertificateSecret` spec causing a fresh reconcile event to create a new secret.
- CRDs have been separated into there own HelmChart due to issue with CRD lifecycle and helm 3

### Added

## [1.0.0] - 2024-09-19

### Added

- **Breaking Change** API spec for both `ExportCertificateSecret` and `ImportCertificateSecret` has been changed to a standard spec for a better usr experience and scalability. `certificateSecretRef` now supports n number of key and secret references from Azure KV or K8s secret as documented in `api/v1alpha1/docs`
- Reconcile loops for termination and normal operation now have an exponential backoff with an upper limit of 30 minutes for better stability.
- Code refactored to follow api changes and a new standard code flow for processing secrets
- Secrets are now purged from AKV by default, can be tuned by user
- Implements new logging functionality

### Bugfix

- Fixes Certificate mount secret for controller manager deployment when using `csiDriverTLSMount` option
- Double reconcile loop occurred when status msg updated, status events for `ExportCertificateSecret` and `ImportCertificateSecret` now filtered from predicate
- Switch to alpine from distroless debian due to hard dependency on openssl

## [0.1.2] - 2024-09-10

- Bugfixes to helmchart including the ability to mount cert secret from CSI driver if cluster dosent have CM installed
- Make scanInterval optional for ImportCertificateSecret specs
- Updates readme with more detail flow

## [0.1.1] - 2024-09-04

### Bugfix

- Fetches the latest version of the object before updating its status
- Fixes issues with Helmchart

## [0.1.0] - 2024-08-28

### Added

- Initial release of external-certificate-operator
