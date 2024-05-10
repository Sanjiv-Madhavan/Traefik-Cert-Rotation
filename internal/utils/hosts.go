package utils

import (
	"fmt"

	muxer "github.com/traefik/traefik/v2/pkg/muxer/http"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
)

// HostCollection allows to aggregate the hosts from ingress resources.
type HostCollection struct {
	hosts map[string]struct{}
}

func NewHostCollection() *HostCollection {
	return &HostCollection{hosts: make(map[string]struct{})}
}

// WithTLSHostsIfAvailable aggregates all hosts found in the provided TLS configuration. If the
// TLS configuration is empty (i.e. `nil`), no hosts are extracted. This method should only be
// called on a freshly initialized aggregator. Takes in Traefik.TLS to extract hosts
func (hc *HostCollection) WithTLSHostsIfAvailable(config *traefik.TLS) *HostCollection {
	if config != nil {
		for _, domain := range config.Domains {
			hc.hosts[domain.Main] = struct{}{}
			for _, san := range domain.SANs {
				hc.hosts[san] = struct{}{}
			}
		}
	}
	return hc
}

// WithRouteHostsIfRequired aggregates all (unique) hosts found in the provided routes. If the
// aggregator already manages at least one host, this method is a noop, regardless of the routes
// passed as parameters.
func (a *HostCollection) WithRouteHostsIfRequired(
	routes []traefik.Route,
) (*HostCollection, error) {
	if len(a.hosts) > 0 {
		return a, nil
	}
	for _, route := range routes {
		if route.Kind == "Rule" {
			hosts, err := muxer.ParseDomains(route.Match)
			if err != nil {
				return nil, fmt.Errorf("failed to parse domains: %s", err)
			}
			for _, host := range hosts {
				a.hosts[host] = struct{}{}
			}
		}
	}
	return a, nil
}

func (a *HostCollection) Len() int {
	return len(a.hosts)
}

// Hosts returns all hosts managed by this aggregator.
func (a *HostCollection) Hosts() []string {
	hosts := make([]string, 0, len(a.hosts))
	for host := range a.hosts {
		hosts = append(hosts, host)
	}
	return hosts
}
