# Kubernetes Deployment Guide

This guide walks through deploying lmon on a Kubernetes cluster with node-level monitoring agents and a centralized aggregator dashboard, optionally exposed via Ingress.

## Architecture

```
                    ┌─────────────────────────┐
                    │        Ingress           │
                    │  (lmon.example.com)      │
                    └───────────┬─────────────┘
                                │
                    ┌───────────▼─────────────┐
                    │    lmon-aggregator       │
                    │    (Deployment, 1 pod)   │
                    │                         │
                    │  - Cluster-wide view     │
                    │  - K8s event monitoring  │
                    │  - K8s node monitoring   │
                    │  - Scrapes node agents   │
                    └───────────┬─────────────┘
                                │ scrapes /metrics
              ┌─────────────────┼─────────────────┐
              │                 │                  │
    ┌─────────▼──────┐ ┌───────▼────────┐ ┌──────▼─────────┐
    │  lmon-node     │ │  lmon-node     │ │  lmon-node     │
    │  (DaemonSet)   │ │  (DaemonSet)   │ │  (DaemonSet)   │
    │                │ │                │ │                │
    │  - Host disk   │ │  - Host disk   │ │  - Host disk   │
    │  - Host CPU    │ │  - Host CPU    │ │  - Host CPU    │
    │  - Host memory │ │  - Host memory │ │  - Host memory │
    └────────────────┘ └────────────────┘ └────────────────┘
       Worker Node 1      Worker Node 2      Worker Node 3
```

**lmon-node** agents run as a DaemonSet (one per node) and monitor the physical host's disk, CPU, and memory. They expose a `/metrics` JSON endpoint.

**lmon-aggregator** runs as a single-replica Deployment. It discovers node agents by pod label, scrapes their `/metrics` endpoints, and presents a unified cluster dashboard. It can also run Kubernetes-native monitors for cluster events and node conditions.

Users access the aggregator dashboard via an Ingress or port-forward.

## Prerequisites

- A Kubernetes cluster (v1.24+)
- `kubectl` configured to access the cluster
- The lmon container image built and available (in a registry or loaded locally)

### Building the Image

```bash
# Build locally
docker build -t lmon:latest .

# Tag and push to a registry (adjust for your registry)
docker tag lmon:latest your-registry.com/lmon:latest
docker push your-registry.com/lmon:latest
```

If using a private registry, update the `image:` field in `daemonset.yaml` and `deployment.yaml` accordingly, and ensure your cluster has pull credentials configured.

## Step 1: Create the Namespace

```bash
kubectl create namespace lmon
```

## Step 2: Apply RBAC

The RBAC manifest creates service accounts and cluster roles for both node agents and the aggregator.

```bash
kubectl apply -f deploy/kubernetes/rbac.yaml
```

This creates:
- `lmon-node` ServiceAccount — minimal permissions (read own node)
- `lmon-aggregator` ServiceAccount — read access to pods, nodes, events, endpoints, services

## Step 3: Apply ConfigMaps

**Node agent config** — monitors the physical host via mounted filesystems:

```bash
kubectl apply -f deploy/kubernetes/configmap-node.yaml
```

The default node config monitors:
- `/hostroot` (the host's root filesystem, mounted read-only)
- `/hostroot/var` (the host's /var partition)
- CPU and memory (via host PID namespace)

**Aggregator config** — Kubernetes-native monitors and node discovery:

```bash
kubectl apply -f deploy/kubernetes/configmap-aggregator.yaml
```

The default aggregator config:
- Enables Kubernetes integration (`kubernetes.enabled: true`)
- Discovers node agents by label `app.kubernetes.io/name=lmon-node`
- Monitors Kubernetes events cluster-wide
- Monitors Kubernetes node conditions

### Customizing the Node Config

Edit `configmap-node.yaml` to add monitors for your environment:

```yaml
data:
  config.yaml: |
    monitoring:
      interval: 30
      disk:
        root:
          path: /hostroot
          threshold: 80
          icon: hdd
        data:
          path: /hostroot/data    # Add additional mount points
          threshold: 90
          icon: hdd
      system:
        cpu:
          threshold: 90
          icon: cpu
        memory:
          threshold: 90
          icon: speedometer
        title: "lmon Node Agent"
      healthcheck:
        app:
          url: http://my-app.default.svc:8080/healthz
          timeout: 5
          icon: heart-pulse
      ping:
        gateway:
          address: 10.0.0.1
          timeout: 1000
          amberThreshold: 50
          icon: wifi
    web:
      host: 0.0.0.0
      port: 8080
```

### Customizing the Aggregator Config

Edit `configmap-aggregator.yaml` to add Kubernetes service monitors:

```yaml
data:
  config.yaml: |
    kubernetes:
      enabled: true
      in_cluster: true
    aggregator:
      node_label: "app.kubernetes.io/name=lmon-node"
      node_port: 8080
      node_metrics_path: /metrics
      scrape_interval: 30
    monitoring:
      interval: 30
      k8sevents:
        default:
          namespaces: ""          # Empty = all namespaces
          threshold: 10
          window: 600
          icon: lightning
        production:
          namespaces: "production"
          threshold: 5
          window: 300
          icon: lightning
      k8snodes:
        cluster:
          icon: hdd-rack
      k8sservice:
        api:
          namespace: production
          service: api-server
          health_path: /healthz
          port: 8080
          threshold: 80
          timeout: 5
          icon: globe
        frontend:
          namespace: production
          service: web-frontend
          health_path: /health
          port: 3000
          threshold: 80
          icon: globe
      system:
        cpu:
          threshold: 90
          icon: cpu
        memory:
          threshold: 90
          icon: speedometer
        title: "lmon Cluster Dashboard"
    web:
      host: 0.0.0.0
      port: 8080
    webhook:
      enabled: true
      url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
```

## Step 4: Deploy Node Agents (DaemonSet)

```bash
kubectl apply -f deploy/kubernetes/daemonset.yaml
```

The DaemonSet:
- Runs one pod per node (including control plane nodes via toleration)
- Mounts the host root filesystem at `/hostroot` (read-only)
- Uses host PID namespace for accurate CPU/memory metrics
- Exposes port 8080 for metrics scraping

Verify nodes are running:

```bash
kubectl -n lmon get pods -l app.kubernetes.io/component=node -o wide
```

Each node agent's metrics endpoint is available at `http://<pod-ip>:8080/metrics`.

## Step 5: Deploy the Aggregator

```bash
kubectl apply -f deploy/kubernetes/deployment.yaml
```

The aggregator:
- Runs as a single-replica Deployment
- Uses `LMON_MODE=aggregator` to enable aggregator mode
- Discovers node agents by pod label
- Provides the cluster-wide dashboard

Verify the aggregator is running:

```bash
kubectl -n lmon get pods -l app.kubernetes.io/component=aggregator
```

## Step 6: Deploy Services

```bash
kubectl apply -f deploy/kubernetes/service.yaml
```

This creates:
- `lmon-aggregator` — ClusterIP service for the aggregator dashboard (port 8080)
- `lmon-node` — Headless service for node agent discovery

### Quick Access via Port Forward

```bash
kubectl -n lmon port-forward svc/lmon-aggregator 8080:8080
```

Then open http://localhost:8080 in your browser.

## Step 7: Expose via Ingress (Optional)

Create an Ingress resource to expose the aggregator dashboard externally. Below are examples for common Ingress controllers.

### nginx Ingress Controller

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: lmon-aggregator
  namespace: lmon
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
    - host: lmon.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: lmon-aggregator
                port:
                  number: 8080
```

### With TLS (cert-manager)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: lmon-aggregator
  namespace: lmon
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - lmon.example.com
      secretName: lmon-tls
  rules:
    - host: lmon.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: lmon-aggregator
                port:
                  number: 8080
```

### Traefik Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: lmon-aggregator
  namespace: lmon
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    traefik.ingress.kubernetes.io/router.tls: "true"
spec:
  rules:
    - host: lmon.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: lmon-aggregator
                port:
                  number: 8080
```

### AWS ALB Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: lmon-aggregator
  namespace: lmon
  annotations:
    kubernetes.io/ingress.class: alb
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTPS":443}]'
    alb.ingress.kubernetes.io/certificate-arn: arn:aws:acm:region:account:certificate/cert-id
spec:
  rules:
    - host: lmon.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: lmon-aggregator
                port:
                  number: 8080
```

Apply your chosen Ingress:

```bash
kubectl apply -f ingress.yaml
```

## Step 8: Add PostgreSQL for History (Optional)

To enable the history page and sparklines, deploy a PostgreSQL instance and configure the database URL.

### Using an Existing PostgreSQL

Add the database URL to the aggregator ConfigMap:

```yaml
database:
  url: "postgres://lmon:password@postgres.lmon.svc:5432/lmon?sslmode=disable"
  retention_days: 7
  batch_size: 1000
  prune_interval: 60
```

### Quick PostgreSQL Deployment

For testing, deploy a simple PostgreSQL instance:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: lmon
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
        - name: postgres
          image: postgres:16-alpine
          env:
            - name: POSTGRES_DB
              value: lmon
            - name: POSTGRES_USER
              value: lmon
            - name: POSTGRES_PASSWORD
              value: lmon-secret  # Use a Secret in production
          ports:
            - containerPort: 5432
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
      volumes:
        - name: data
          emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: lmon
spec:
  selector:
    app: postgres
  ports:
    - port: 5432
      targetPort: 5432
```

Then update the aggregator ConfigMap to include the database section and re-apply.

## All-in-One Quick Start

```bash
# Create namespace
kubectl create namespace lmon

# Apply all manifests
kubectl apply -f deploy/kubernetes/

# Wait for pods to be ready
kubectl -n lmon rollout status daemonset/lmon-node
kubectl -n lmon rollout status deployment/lmon-aggregator

# Access the dashboard
kubectl -n lmon port-forward svc/lmon-aggregator 8080:8080
# Open http://localhost:8080
```

## Updating Configuration

ConfigMap changes require a pod restart to take effect:

```bash
# After editing a ConfigMap
kubectl -n lmon rollout restart daemonset/lmon-node
kubectl -n lmon rollout restart deployment/lmon-aggregator
```

## Troubleshooting

### Node agents not discovered

- Verify node pods are running: `kubectl -n lmon get pods -l app.kubernetes.io/name=lmon-node`
- Check the aggregator logs: `kubectl -n lmon logs deployment/lmon-aggregator`
- Ensure the `node_label` in the aggregator config matches the DaemonSet pod labels
- Verify RBAC allows the aggregator to list pods: `kubectl auth can-i list pods --as=system:serviceaccount:lmon:lmon-aggregator`

### Disk monitors show container-only values

- Ensure the DaemonSet mounts the host root filesystem at `/hostroot`
- Ensure `hostPID: true` is set on the DaemonSet pod spec
- Verify disk paths in the ConfigMap point to `/hostroot/...` not `/...`

### Metrics endpoint not responding

- Check the node pod logs: `kubectl -n lmon logs <pod-name>`
- Verify the pod is healthy: `kubectl -n lmon exec <pod-name> -- wget -qO- http://localhost:8080/healthz`
- Check the headless service resolves: `kubectl -n lmon run -it --rm debug --image=busybox -- nslookup lmon-node.lmon.svc`

### Aggregator shows "No nodes discovered"

- The aggregator scrapes on a schedule (default 30 seconds). Wait for the first scrape cycle.
- Check aggregator logs for discovery errors.
- Verify the aggregator service account has permission to list pods.

## Uninstalling

```bash
kubectl delete namespace lmon
kubectl delete clusterrole lmon-aggregator lmon-node
kubectl delete clusterrolebinding lmon-aggregator lmon-node
```
