package controllers

import (
	"fmt"

	"github.com/Sanjiv-Madhavan/Traefik-Cert-Rotation/internal/config"
	"github.com/Sanjiv-Madhavan/Traefik-Cert-Rotation/internal/integrations"
	"github.com/Sanjiv-Madhavan/Traefik-Cert-Rotation/internal/k8s"
	"github.com/Sanjiv-Madhavan/Traefik-Cert-Rotation/internal/utils"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

func integrationsFromConfig(
	config config.Config, client client.Client,
) ([]integrations.Integration, error) {
	result := make([]integrations.Integration, 0)
	externalDNS := config.Integrations.ExternalDNS
	if config.Integrations.ExternalDNS != nil {
		if (externalDNS.TargetService == nil) == (len(externalDNS.TargetIPs) == 0) {
			return nil, fmt.Errorf(
				"exactly one of `targetService` and `targetIPs` must be set for external-dns",
			)
		}
		if externalDNS.TargetService != nil {
			result = append(result, integrations.NewExternalDNS(
				client, utils.NewServiceTarget(
					externalDNS.TargetService.Name,
					externalDNS.TargetService.Namespace,
				),
			))
		} else {
			result = append(result, integrations.NewExternalDNS(
				client, utils.NewStaticTarget(externalDNS.TargetIPs...),
			))
		}
	}

	certManager := config.Integrations.CertManager
	if certManager != nil {
		result = append(result, integrations.NewCertManager(client, certManager.Template))
	}
	return result, nil
}

func builderWithIntegrations(
	builder *builder.Builder,
	integrations []integrations.Integration,
	ctrlClient client.Client,
	logger *zap.Logger,
) *builder.Builder {
	// Reconcile whenever an owned resource of one of the integrations is modified
	for _, itg := range integrations {
		builder = builder.Owns(itg.OwnedResource())
	}

	// watch for dependent resources
	for _, itg := range integrations {
		if itg.WatchedObject() != nil {
			var list traefik.IngressRouteList
			enqueueFunc := k8s.EnqueueMapFunc(ctrlClient, logger, itg.WatchedObject(), &list,
				func(list *traefik.IngressRouteList) []client.Object {
					return utils.Map(list.Items, func(v traefik.IngressRoute) client.Object {
						return &v
					})
				},
			)
			builder = builder.Watches(
				itg.WatchedObject(),
				handler.EnqueueRequestsFromMapFunc(enqueueFunc),
			)
		}
	}
	return builder
}
