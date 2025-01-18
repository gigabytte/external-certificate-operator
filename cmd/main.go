package main

import (
	"crypto/tls"
	"flag"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	certdistributionv1alpha1 "github.com/gigabytte/external-certificate-operator/api/v1alpha1"
	"github.com/gigabytte/external-certificate-operator/internal/controller"
	"github.com/gigabytte/external-certificate-operator/internal/log"
)

var (
	scheme                       = runtime.NewScheme()
	setupLog                     = log.WithName("setup")
	allowedImportCrossNamespaces string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(certdistributionv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection,
		"leader-elect",
		false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.", // nolint:lll
	)
	flag.BoolVar(&secureMetrics, "metrics-secure", false, "If set the metrics endpoint is served securely")
	flag.BoolVar(&enableHTTP2, "enable-http2", false, "If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.StringVar(&allowedImportCrossNamespaces,
		"allowed-import-cross-namespaces",
		"",
		"Comma-separated list of allowed namespaces that a given ImportCertficateSecret object can create secret in outside of its native namespace (ie. Cross ns secret creation)", // nolint:lll
	)

	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	tlsOpts := getTLSOptions(enableHTTP2)
	webhookServer := webhook.NewServer(webhook.Options{TLSOpts: tlsOpts})

	mgr, err := createManager(metricsAddr, secureMetrics, probeAddr, enableLeaderElection, tlsOpts, webhookServer)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = setupControllers(mgr); err != nil {
		setupLog.Error(err, "unable to set up controllers")
		os.Exit(1)
	}

	if err = setupWebhooks(mgr); err != nil {
		setupLog.Error(err, "unable to set up webhooks")
		os.Exit(1)
	}

	setupHealthChecks(mgr)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func getTLSOptions(enableHTTP2 bool) []func(*tls.Config) {
	if !enableHTTP2 {
		return []func(*tls.Config){func(c *tls.Config) {
			setupLog.Info("disabling http/2")
			c.NextProtos = []string{"http/1.1"}
		}}
	}
	return nil
}

// nolint:lll
func createManager(metricsAddr string, secureMetrics bool, probeAddr string, enableLeaderElection bool, tlsOpts []func(*tls.Config), webhookServer webhook.Server) (ctrl.Manager, error) {
	return ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "140f7039.external-certificate.io",
	})
}

func setupControllers(mgr ctrl.Manager) error {
	if err := (&controller.ExportCertificateSecretReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	allowedImportCrossNamespacesList := strings.Split(allowedImportCrossNamespaces, ",")
	if len(allowedImportCrossNamespacesList) > 0 {
		allowedImportCrossNamespacesList = cleanNamespaces(allowedImportCrossNamespacesList)
		setupLog.Info("Allowed namespaces for ImportCertificateSecret", "namespaces", allowedImportCrossNamespacesList)
	}
	certdistributionv1alpha1.SetAllowedNamespaces(allowedImportCrossNamespacesList)

	if err := (&controller.ImportCertificateSecretReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}

func setupWebhooks(mgr ctrl.Manager) error {
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		setupLog.Info("Setting up webhook for ExportCertificateSecret")
		if err := (&certdistributionv1alpha1.ExportCertificateSecret{}).SetupWebhookWithManager(mgr); err != nil {
			return err
		}

		setupLog.Info("Setting up webhook for ImportCertificateSecret")
		if err := (&certdistributionv1alpha1.ImportCertificateSecret{}).SetupWebhookWithManager(mgr); err != nil {
			return err
		}
	}
	return nil
}

func setupHealthChecks(mgr ctrl.Manager) {
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
}

// cleanNamespaces trims and cleans all values in the list.
func cleanNamespaces(namespaces []string) []string {
	cleanedNamespaces := make([]string, len(namespaces))
	for i, ns := range namespaces {
		cleanedNamespaces[i] = strings.ReplaceAll(strings.TrimSpace(strings.Trim(ns, `"`)), `\`, "")
	}
	return cleanedNamespaces
}
