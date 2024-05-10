package controllers

import (
	"context"
	"fmt"

	"github.com/Sanjiv-Madhavan/Traefik-Cert-Rotation/internal/config"
	"github.com/Sanjiv-Madhavan/Traefik-Cert-Rotation/internal/integrations"
	"github.com/Sanjiv-Madhavan/Traefik-Cert-Rotation/internal/utils"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	"go.uber.org/zap"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IngressRouteReconciler struct {
	client.Client
	logger       *zap.Logger
	selector     utils.Selector
	integrations []integrations.Integration
}

func NewIngressRouteReconciler(
	client client.Client, logger *zap.Logger, config config.Config,
) (IngressRouteReconciler, error) {
	integrations, err := integrationsFromConfig(config, client)
	if err != nil {
		return IngressRouteReconciler{}, fmt.Errorf("failed to initialize integrations: %s", err)
	}
	return IngressRouteReconciler{
		Client:       client,
		logger:       logger,
		selector:     utils.NewSelector(config.Selector.IngressClass),
		integrations: integrations,
	}, nil
}

func (r *IngressRouteReconciler) Reconcile(
	ctx context.Context, req ctrl.Request,
) (ctrl.Result, error) {

	logger := r.logger.With(zap.String("name", req.String()))

	// First, we retrieve the full resource
	var ingressRoute traefik.IngressRoute

	if err := r.Get(ctx, req.NamespacedName, &ingressRoute); err != nil {
		if !apierrs.IsNotFound(err) {
			logger.Error("unable to query for ingress route", zap.Error(err))
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Then, we check if the resource should be processed
	// ignore annotation must not be there in ingressRoute resource
	// Check while debugging
	if !r.selector.Matches(ingressRoute.Annotations) {
		logger.Debug("ignoring ingress route")
		return ctrl.Result{}, nil
	}
	logger.Debug("reconciling ingress route")

	// Now, we have to ensure that all the dependent resources exist by calling all integrations.
	// For this, we first have to extract information about the ingress.

	// Check Why are we aggregating routeHosts if no TLS is present
	hostCollection, err := utils.NewHostCollection().
		WithTLSHostsIfAvailable(ingressRoute.Spec.TLS).
		WithRouteHostsIfRequired(ingressRoute.Spec.Routes)
	if err != nil {
		logger.Error("failed to parse hosts from ingress route", zap.Error(err))
		return ctrl.Result{}, err
	}

	info := integrations.IngressInfo{
		Hosts: hostCollection.Hosts(),
		TLSSecretName: utils.AndThen(ingressRoute.Spec.TLS, func(tls traefik.TLS) string {
			return tls.SecretName
		}),
	}

	// Run integrations. We needed ingress info to pass in to create/update resource
	for _, integration := range r.integrations {
		// Check while debugging
		// "github.sanjivmadhavan.io/ignore" label must have cert-manager and external-dns
		if !r.selector.MatchesIntegration(ingressRoute.Annotations, integration.Name()) {
			// If integration is ignored, skip it
			logger.Debug("ignoring integration", zap.String("integration", integration.Name()))
			continue
		}
		if err := integration.UpdateResource(ctx, &ingressRoute, info); err != nil {
			logger.Error("failed to upsert resource",
				zap.String("integration", integration.Name()), zap.Error(err),
			)
			return ctrl.Result{}, err
		}
		logger.Debug("successfully upserted resource", zap.String("integration", integration.Name()))
	}
	logger.Info("ingress route is up to date")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).For(&traefik.IngressRoute{})
	builder = builderWithIntegrations(builder, r.integrations, r, r.logger)
	return builder.Complete(r)
}
