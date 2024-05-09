package config

import (
	v1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
)

type Config struct {
	ControllerConfig `json:",inline"`
	Selector         IngressSelector    `json:"selector"`
	Integrations     IntegrationConfigs `json:"integrations"`
}

type ControllerConfig struct {
	Health         HealthConfig         `json:"health,omitempty"`
	LeaderElection LeaderElectionConfig `json:"leaderElection,omitempty"`
	Metrics        MetricsConfig        `json:"metrics,omitempty"`
}

type HealthConfig struct {
	HealthProbeBindAddress string `json:"healthProbeBindAddress,omitempty"`
}

type LeaderElectionConfig struct {
	LeaderElect       bool   `json:"leaderElect,omitempty"`
	ResourceName      string `json:"resourceName,omitempty"`
	ResourceNamespace string `json:"resourceNamespace,omitempty"`
}

// MetricsConfig provides configuration for the controller metrics.
type MetricsConfig struct {
	BindAddress string `json:"bindAddress,omitempty"`
}

// IngressSelector can be used to limit operations to ingresses with a specific ingressClass.
type IngressSelector struct {
	IngressClass *string `json:"ingressClass,omitempty"`
}

// IntegrationConfigs describes the configurations for all integrations. using pointers for optional configurations
// allows for clearer indication of whether a configuration is present or absent, making the code more explicit and
// easier to understand. If the CertManager configuration is optional and pointers are used, omitting the CertManager
// configuration would result in a nil value for the CertManager pointer, indicating that it is not configured, rather
// than causing a compile-time or runtime error.
type IntegrationConfigs struct {
	ExternalDNS *ExternalDNSIntegrationConfig `json:"externalDNS"`
	CertManager *CertManagerIntegrationConfig `json:"certManager"`
}

// ExternalDNSIntegrationConfig describes the configuration for the external-dns integration.
// Exactly one of target and target IPs should be set.
type ExternalDNSIntegrationConfig struct {
	TargetService *ServiceRef `json:"targetService,omitempty"`
	TargetIPs     []string    `json:"targetIPs,omitempty"`
}

type CertManagerIntegrationConfig struct {
	Template v1.Certificate `json:"certificateTemplate"`
}

// ServiceRef uniquely describes a Kubernetes service.
type ServiceRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}
