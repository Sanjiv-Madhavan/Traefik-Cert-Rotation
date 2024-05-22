## What is Traefik?
#### *P.S. It has a very cute [logo](https://raw.githubusercontent.com/docker-library/docs/a6cc2c5f4bc6658168f2a0abbb0307acaefff80e/traefik/logo.png) <3*

Traefik is an open-source Edge Router that makes publishing your services a fun and easy experience. It receives requests on behalf of your system and finds out which components are responsible for handling them.

![Traefik](https://doc.traefik.io/traefik/assets/img/traefik-concepts-1.png)
source: [Traefik](https://doc.traefik.io/traefik)

With Traefik, there is no need to maintain and synchronize a separate configuration file: everything happens automatically, in real time (no restarts, no connection interruptions). With Traefik, you spend time developing and deploying new features to your system, not on configuring and maintaining its working state.

## Problem statement:

[Traefik documentation](https://doc.traefik.io/traefik/providers/kubernetes-ingress/#ingressclass) says:

> By design, Traefik is a stateless application, meaning that it only derives its configuration from the environment it runs in, without additional configuration. For this reason, users can run multiple instances of Traefik at the same time to achieve HA, as is a common pattern in the kubernetes ecosystem.
>
> When using a single instance of Traefik Proxy with Let's Encrypt, you should encounter no issues. However, this could be a single point of failure. Unfortunately, it is not possible to run multiple instances of Traefik 2.0 with Let's Encrypt enabled, because there is no way to ensure that the correct instance of Traefik receives the challenge request, and subsequent responses. Previous versions of Traefik used a KV store to attempt to achieve this, but due to sub-optimal performance that feature was dropped in 2.0.
>
> If you need Let's Encrypt with high availability in a Kubernetes environment, we recommend using Traefik Enterprise which includes distributed Let's Encrypt as a supported feature.
>
> If you want to keep using Traefik Proxy, LetsEncrypt HA can be achieved by using a Certificate Controller such as Cert-Manager. When using Cert-Manager to manage certificates, it creates secrets in your namespaces that can be referenced as TLS secrets in your ingress objects.

## Solution:

### Offloading Let's Encrypt challenges to cert-manager

Cert-manager is a Kubernetes add-on that helps automate the management and issuance of TLS certificates from various certificate authorities, including Let's Encrypt. It solves the issue of running multiple instances of Traefik 2.0 with Let's Encrypt enabled by taking over the responsibility of handling the certificate issuance and renewal process.

Here's how cert-manager works and how it addresses the problem:

1. **Certificate Management**: Cert-manager watches for Kubernetes resources such as Ingress objects that specify TLS certificates. When it detects a new or updated resource, it automatically requests the necessary certificates from the configured certificate authorities.
2. **Issuance and Renewal**: Cert-manager communicates with the Let's Encrypt ACME (Automated Certificate Management Environment) server to request TLS certificates. It handles the entire process of domain validation, certificate issuance, and renewal automatically.
3. **Storing Certificates**: Once the certificates are issued, cert-manager stores them securely as Kubernetes secrets within the cluster.
4. **Automatic Renewal**: Cert-manager monitors the expiration dates of certificates it manages and automatically initiates renewal processes before they expire. This ensures that your applications always have valid TLS certificates.

By leveraging cert-manager to manage TLS certificates for your Traefik instances, you no longer rely on Traefik itself to handle Let's Encrypt challenges and certificate management. Instead, cert-manager handles these tasks independently of Traefik. This decoupling allows you to run multiple instances of Traefik without worrying about conflicts or challenges related to Let's Encrypt issuance and renewal. Each instance of Traefik can then reference the TLS certificates stored as Kubernetes secrets managed by cert-manager, ensuring high availability and proper TLS encryption across your Kubernetes environment.

## Installation:
```bash
helm install traefik-cert-rotation ./traefik-cert-rotation --namespace traefik-cert-rotation --create-namespace
```

## Process
Traefik-cert-rotater processes Traefik IngressRoute resources. Let's assume we have the following ingress route which forwards requests to an Nginx backend:

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: my-ingress
spec:
  routes:
    - kind: Rule
      match: Host(`www.example.com`) && PathPrefix(`/images`)
      services:
        - name: nginx
  tls:
    secretName: www-tls-certificate
```
Traefik-cert-rotater now automatically extracts information from the ingress route object:

1. #### Host and Path Matching:

- The ingress route is concerned with a single host, www.example.com.
It forwards requests to this host that match the specified path prefix, /images.

2. #### Service Specification:

- Requests matching the host and path prefix are forwarded to the service named nginx. This service is responsible for handling the incoming requests.

3. #### TLS Configuration:

- The ingress route specifies that requests should be TLS-protected. This means that the communication should be encrypted to ensure security.
- A TLS certificate is specified, which is stored in the Kubernetes secret named www-tls-certificate. This certificate is used to establish a secure HTTPS connection for requests to www.example.com.

Once Traefik-cert-rotater processes this ingress route object, it extracts the above information and distributes it to all configured integrations. This ensures that all components in the system are aware of the routing rules, the service handling the requests, and the TLS requirements. This automated extraction and dissemination streamline the management of ingress routes and enhance the security and efficiency of handling requests.

## Integrations
Integrations are entirely independent of each other. Enabling an integration causes Traefik-cert-rotater to generate an integration-specific resource (typically a CRD) for each ingress route that it processes.

#### Cert-Manager
The cert-manager integration allows Traefik-cert-rotater to create a Certificate resource for an IngressRoute if the ingress (1) specifies .spec.tls.secretName and (2) references at least one host. Using the example ingress route from above, Traefik-cert-rotater creates the following resource:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  # The name is automatically generated from the name of the ingress route.
  name: my-ingress-tls
  labels:
    kubernetes.io/managed-by: Traefik-cert-rotater
spec:
  # The issuer reference is obtained from the configuration of the cert-manager integration.
  issuerRef:
    kind: ClusterIssuer
    name: ca-issuer
  dnsNames:
    - www.example.com
  secretName: www-tls-certificate
  ```

#### External-DNS
The external-dns integration causes Traefik-cert-rotater to create a DNSEndpoint resource for an IngressRoute if the ingress references at least one host. Given the example ingress route above, Traefik-cert-rotater creates the following endpoint:

```yaml
apiVersion: externaldns.k8s.io/v1alpha1
kind: DNSEndpoint
metadata:
  # The name is the same as the ingress's name.
  name: my-ingress
  labels:
    kubernetes.io/managed-by: Traefik-cert-rotater
spec:
  endpoints:
    - dnsName: www.example.com
      recordTTL: 300
      recordType: A
      targets:
        # The target is the public (or, if unavailable, private) IP address of your Traefik
        # instance. The Kubernetes service to source the IP from is obtained from the configuration
        # of the external-dns integration.
        - 10.96.0.10```