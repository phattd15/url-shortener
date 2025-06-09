# Kubernetes Deployment Guide for URL Shortener

This guide provides instructions for deploying a high-performance URL shortener service with PostgreSQL and Redis master-slave configurations on Kubernetes.

## Architecture Overview

- **Application**: 3 replicas of URL shortener service
- **PostgreSQL**: 1 master + 2 slave replicas
- **Redis**: 1 master + 2 slave replicas
- **Ingress**: NGINX ingress controller for external access

## Prerequisites

- Docker
- kubectl
- kind (will be installed automatically)
- 8GB+ RAM recommended
- 20GB+ disk space

## Quick Start

### 1. Automated Setup

Run the setup script to create everything automatically:

```bash
./scripts/setup-k8s.sh setup
```

This will:
- Install kind if needed
- Create a local Docker registry
- Set up a 3-node Kubernetes cluster
- Install NGINX ingress controller
- Build and push the application image
- Deploy PostgreSQL master-slave setup
- Deploy Redis master-slave setup
- Deploy the URL shortener application
- Show access information

### 2. Access the Application

After deployment, add to your `/etc/hosts`:
```
127.0.0.1 url-shortener.local
```

Then access:
- **Application**: http://url-shortener.local
- **Swagger Docs**: http://url-shortener.local/swagger/index.html

Or use port-forwarding:
```bash
kubectl port-forward -n url-shortener service/url-shortener-service 8080:80
```

## Manual Deployment Steps

### 1. Create Kubernetes Cluster

```bash
# Install kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-darwin-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Create cluster
kind create cluster --name url-shortener-cluster
```

### 2. Deploy Components

```bash
# Create namespace
kubectl apply -f manifests/namespace.yaml

# Deploy PostgreSQL master-slave
kubectl apply -f manifests/postgres/

# Deploy Redis master-slave
kubectl apply -f manifests/redis/

# Build and load application image
docker build -t url-shortener:latest .
kind load docker-image url-shortener:latest --name url-shortener-cluster

# Deploy application
kubectl apply -f manifests/app/
```

### 3. Install Ingress Controller

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

# Wait for ingress controller
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s
```

## Management Commands

### Check Status

```bash
./scripts/setup-k8s.sh status
```

### Rebuild Application

```bash
./scripts/setup-k8s.sh rebuild
```

### Cleanup Everything

```bash
./scripts/setup-k8s.sh cleanup
```

## Architecture Details

### PostgreSQL Master-Slave Setup

- **Master**: 1 replica for writes
  - Persistent storage: 10Gi
  - Replication enabled
- **Slaves**: 2 replicas for reads
  - Streaming replication from master
  - Read-only mode

### Redis Master-Slave Setup

- **Master**: 1 replica for writes
  - Persistent storage: 5Gi
  - AOF enabled
- **Slaves**: 2 replicas for reads
  - Replication from master
  - No AOF (memory optimization)

### Application Configuration

The application connects to:
- **Database**: postgres-master (writes), postgres-slave (reads)
- **Cache**: redis-master (writes), redis-slave (reads)

Environment variables:
```yaml
DB_HOST: postgres-master
DB_PORT: 5432
DB_USER: postgres
DB_PASSWORD: password
DB_NAME: urlshortener
REDIS_ADDR: redis-master:6379
REDIS_PASSWORD: ""
REDIS_DB: 0
PORT: 8080
GIN_MODE: release
```

## Monitoring and Debugging

### View Logs

```bash
# Application logs
kubectl logs -n url-shortener deployment/url-shortener -f

# PostgreSQL master logs
kubectl logs -n url-shortener statefulset/postgres-master -f

# Redis master logs
kubectl logs -n url-shortener statefulset/redis-master -f
```

### Check Pod Status

```bash
kubectl get pods -n url-shortener -o wide
```

### Check Services

```bash
kubectl get services -n url-shortener
```

### Access Redis CLI

```bash
kubectl exec -it -n url-shortener redis-master-0 -- redis-cli
```

### Access PostgreSQL

```bash
kubectl exec -it -n url-shortener postgres-master-0 -- psql -U postgres -d urlshortener
```

## Testing the Deployment

### Basic API Tests

```bash
# Create a short URL
curl -X POST http://url-shortener.local/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/very/long/url"}'

# Test redirection
curl -L http://url-shortener.local/abc123

# Check statistics
curl http://url-shortener.local/stats/abc123

# Health check
curl http://url-shortener.local/health
```

### Load Testing

```bash
# Install Apache Bench (if not available)
# macOS: brew install httpie
# Ubuntu: apt-get install apache2-utils

# Load test URL creation
ab -n 1000 -c 10 -T 'application/json' \
  -p <(echo '{"url": "https://example.com/test"}') \
  http://url-shortener.local/shorten

# Load test redirection
ab -n 1000 -c 10 http://url-shortener.local/abc123
```

## Scaling

### Scale Application

```bash
kubectl scale deployment url-shortener --replicas=5 -n url-shortener
```

### Scale Redis Slaves

```bash
kubectl scale statefulset redis-slave --replicas=3 -n url-shortener
```

### Scale PostgreSQL Slaves

```bash
kubectl scale statefulset postgres-slave --replicas=3 -n url-shortener
```

## Production Considerations

### Security
- Use secrets for passwords instead of plain text
- Enable TLS for all communications
- Set up network policies
- Use service accounts with minimal permissions

### High Availability
- Deploy across multiple nodes/zones
- Use pod disruption budgets
- Configure anti-affinity rules

### Persistence
- Use persistent volumes with backup
- Configure storage classes for different performance needs
- Set up regular database backups

### Monitoring
- Deploy Prometheus and Grafana
- Set up alerts for service health
- Monitor database and cache performance

### Ingress
- Use cert-manager for TLS certificates
- Configure rate limiting
- Set up external DNS for production domains

## Troubleshooting

### Common Issues

1. **Pods stuck in Pending**
   - Check resource limits
   - Verify persistent volume claims

2. **Application can't connect to database**
   - Check service names and ports
   - Verify database is ready

3. **Ingress not working**
   - Check ingress controller status
   - Verify /etc/hosts entry

4. **Out of disk space**
   - Clean up unused images: `docker system prune`
   - Increase cluster disk size

### Debug Commands

```bash
# Describe problematic pod
kubectl describe pod <pod-name> -n url-shortener

# Check events
kubectl get events -n url-shortener --sort-by='.lastTimestamp'

# Check resource usage
kubectl top pods -n url-shortener
kubectl top nodes

# Check persistent volumes
kubectl get pv,pvc -n url-shortener
```

## Alternative: Helm Deployment

For a more flexible deployment, you can use the provided Helm chart:

```bash
# Add Bitnami repo for dependencies
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Install with Helm
helm install url-shortener ./helm/

# Upgrade
helm upgrade url-shortener ./helm/

# Uninstall
helm uninstall url-shortener
```

The Helm chart provides more configuration options and easier management of the entire stack. 