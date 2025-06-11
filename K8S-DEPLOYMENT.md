# SavannaCart Kubernetes Deployment

This directory contains Helm charts and deployment scripts for deploying SavannaCart to Kubernetes.

## Prerequisites

- Kubernetes cluster (minikube for local development)
- Helm 3.x
- kubectl
- Docker
- (Optional) kubeseal for sealed secrets

## Quick Start with Minikube

1. **Start minikube:**
   ```powershell
   minikube start
   ```

2. **Deploy SavannaCart:**
   ```powershell
   .\deploy-k8s.ps1 -Environment development -InstallDependencies
   ```

3. **Access the application:**
   ```powershell
   # Enable ingress
   minikube addons enable ingress
   
   # Get minikube IP
   $minikubeIP = minikube ip
   
   # Add to hosts file (run as administrator)
   Add-Content -Path "C:\Windows\System32\drivers\etc\hosts" -Value "$minikubeIP api.savannacart.local"
   
   # Access the API
   curl http://api.savannacart.local/v1/api/healthcheck
   ```

   Or use port-forwarding:
   ```powershell
   kubectl port-forward -n savannacart service/savannacart 8080:80
   # Then access: http://localhost:8080/v1/api/healthcheck
   ```

## Architecture

The Helm chart deploys:

- **SavannaCart API**: Go application (2 replicas by default)
- **PostgreSQL**: Database for persistent storage
- **ConfigMap**: Non-sensitive configuration
- **Secrets**: Sensitive configuration (supports sealed secrets)
- **Ingress**: External access with TLS
- **HPA**: Horizontal Pod Autoscaler for scaling

## Configuration

### Environment Variables

The application uses the following environment variables:

**Non-sensitive (ConfigMap):**
- `APP_NAME`, `APP_VERSION`, `APP_AUTHOR`
- `ENV`, `PORT`
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_SSLMODE`
- `CORS_TRUSTED_ORIGINS`
- `RATE_LIMIT_*`

**Sensitive (Secrets):**
- `DB_USER`, `DB_PASSWORD`
- `JWT_SECRET`
- `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`
- `SMTP_*`
- `SMS_API_*`

### Helm Values

Key configuration options in `values.yaml`:

```yaml
# Applicatiion
app:
  environment: "production"  # or "development"

# Image
image:
  repository: savannacart/api
  tag: latest

# Scaling
replicaCount: 2
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10

# Database
postgresql:
  enabled: true
  auth:
    database: "savannacart"
    username: "savannacart"

# Security
sealedSecrets:
  enabled: false  # Set to true for production
```

## Security with Sealed Secrets

For production deployments, use sealed secrets:

1. **Install Sealed Secrets Controller:**
   ```powershell
   kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.0/controller.yaml
   ```

2. **Generate sealed secrets:**
   ```powershell
   .\generate-sealed-secrets.ps1 -Namespace savannacart
   ```

3. **Update Helm values:**
   ```yaml
   sealedSecrets:
     enabled: true
   ```

4. **Deploy with sealed secrets:**
   ```powershell
   .\deploy-k8s.ps1 -Environment production -UseSealedSecrets
   ```

## Manual Deployment

If you prefer manual deployment:

1. **Build and load Docker image:**
   ```powershell
   docker build -t savannacart/api:latest .
   minikube image load savannacart/api:latest  # For minikube
   ```

2. **Create namespace:**
   ```powershell
   kubectl create namespace savannacart
   ```

3. **Deploy with Helm:**
   ```powershell
   helm upgrade --install savannacart ./savannacart \
     --namespace savannacart \
     --set image.repository=savannacart/api \
     --set image.tag=latest
   ```

## Monitoring and Debugging

### Check deployment status:
```powershell
kubectl get pods -n savannacart
kubectl get services -n savannacart
kubectl get ingress -n savannacart
```

### View logs:
```powershell
kubectl logs -n savannacart deployment/savannacart -f
```

### Debug pod issues:
```powershell
kubectl describe pod -n savannacart <pod-name>
```

### Test database connection:
```powershell
kubectl exec -n savannacart deployment/savannacart-postgresql -- psql -U savannacart -d savannacart -c "SELECT version();"
```

## Customization

### Custom Values File

Create a custom values file for your environment:

```yaml
# values-production.yaml
app:
  environment: "production"

image:
  repository: myregistry/savannacart/api
  tag: "v1.0.0"

ingress:
  hosts:
    - host: api.mycompany.com
  tls:
    - secretName: mycompany-tls
      hosts:
        - api.mycompany.com

postgresql:
  auth:
    password: "secure-production-password"

sealedSecrets:
  enabled: true
```

Deploy with custom values:
```powershell
helm upgrade --install savannacart ./savannacart \
  --namespace savannacart \
  --values values-production.yaml
```

### Environment-specific Deployments

**Development:**
```powershell
.\deploy-k8s.ps1 -Environment development
```

**Staging:**
```powershell
.\deploy-k8s.ps1 -Environment staging -Namespace savannacart-staging
```

**Production:**
```powershell
.\deploy-k8s.ps1 -Environment production -UseSealedSecrets
```

## Uninstall

To remove the deployment:

```powershell
# Remove Helm release
helm uninstall savannacart -n savannacart

# Remove namespace
kubectl delete namespace savannacart

# Remove PVCs (if needed)
kubectl delete pvc --all -n savannacart
```

## Troubleshooting

### Common Issues

1. **Image pull errors:**
   - For minikube: Ensure image is loaded with `minikube image load`
   - For clusters: Check image registry and pull secrets

2. **Database connection issues:**
   - Check PostgreSQL pod status: `kubectl get pods -n savannacart`
   - Verify database credentials in secrets

3. **Ingress not working:**
   - Enable ingress addon: `minikube addons enable ingress`
   - Check ingress controller: `kubectl get pods -n ingress-nginx`

4. **Sealed secrets not decrypting:**
   - Ensure sealed-secrets controller is running
   - Verify namespace and secret name match during encryption

### Health Checks

The application includes health check endpoints:
- Liveness: `/v1/api/healthcheck`
- Readiness: `/v1/api/healthcheck`

## Production Considerations

1. **Resource Limits**: Set appropriate CPU/memory limits
2. **Persistent Storage**: Use proper storage classes for PostgreSQL
3. **Backup Strategy**: Implement database backup solutions
4. **Monitoring**: Add Prometheus/Grafana monitoring
5. **Logging**: Centralized logging with ELK stack
6. **Security**: Use sealed secrets, network policies, and RBAC
