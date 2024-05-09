package main

import (
	"context"
	"flag"
	"os"

	"github.com/Sanjiv-Madhavan/Traefik-Cert-Rotation/internal/config"
	"github.com/borchero/zeus/pkg/zeus"
	certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/external-dns/endpoint"

	"sigs.k8s.io/yaml"
)

func main() {
	var cfgFile string
	flag.StringVar(&cfgFile, "config", "/Users/i513687/Documents/GitHub/Sanjivmadhavan/switchboard/dev/config.yaml", "The config file to use.")
	flag.Parse()

	// Initialize logger
	ctx := context.Background()
	logger := zeus.Logger(ctx)
	defer zeus.Sync()

	// Load the config file if available
	var config config.Config
	if cfgFile != "" {
		contents, err := os.ReadFile(cfgFile)
		if err != nil {
			logger.Fatal("failed to read config file", zap.Error(err))
		}
		if err := yaml.Unmarshal(contents, &config); err != nil {
			logger.Fatal("failed to parse config file", zap.Error(err))
		}
	}

	// Initialize the options and the schema
	options := ctrl.Options{
		Scheme:                  runtime.NewScheme(),
		LeaderElection:          config.LeaderElection.LeaderElect,
		LeaderElectionID:        config.LeaderElection.ResourceName,
		LeaderElectionNamespace: config.LeaderElection.ResourceNamespace,
		Metrics: server.Options{
			BindAddress: config.Metrics.BindAddress,
		},
		HealthProbeBindAddress: config.Health.HealthProbeBindAddress,
	}
	initScheme(config, options.Scheme)
}

func initScheme(config config.Config, scheme *runtime.Scheme) {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(traefik.AddToScheme(scheme))

	if config.Integrations.CertManager != nil {
		utilruntime.Must(certmanager.AddToScheme(scheme))
	}

	if config.Integrations.ExternalDNS != nil {
		groupVersion := schema.GroupVersion{Group: "externaldns.k8s.io", Version: "v1alpha1"}
		scheme.AddKnownTypes(groupVersion,
			&endpoint.DNSEndpoint{},
			&endpoint.DNSEndpointList{},
		)
		metav1.AddToGroupVersion(scheme, groupVersion)
	}
}
