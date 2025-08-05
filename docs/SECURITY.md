# SavannaCart Sealed Secrets Security Guidelines

## 🛡️ SECURITY BEST PRACTICES

### ❌ NEVER DO THIS:
1. Store plaintext secrets in Git repository
2. Put secrets in scripts that get committed
3. Use real production secrets in development
4. Share sealed secret private keys

### ✅ SECURE WORKFLOW:

#### For Development:
1. Use secure-sealed-secrets.sh with dummy/test values
2. Keep .env files in .gitignore
3. Use different credentials for each environment

#### For Production:
1. Use CI/CD pipeline with secret management
2. Store secrets in vault (HashiCorp Vault, AWS Secrets Manager)
3. Rotate secrets regularly
4. Use service accounts with minimal permissions

### 🔄 SECRET ROTATION PROCESS:

```bash
# 1. Generate new sealed secrets
./secure-sealed-secrets.sh production

# 2. Apply new secrets
kubectl apply -f production-sealed-secrets.yaml

# 3. Restart application to pick up new secrets
kubectl rollout restart deployment/savannacart-release -n production

# 4. Verify deployment
kubectl rollout status deployment/savannacart-release -n production

# 5. Clean up old secrets and files
rm production-sealed-secrets.yaml
```

### 🗂️ GITIGNORE ADDITIONS:
```
# Secrets and sensitive files
*.env
*-sealed-secrets.yaml
complete-sealed-secrets.yaml
create-complete-secrets.sh

# Only commit the template and secure generator
savannacart/templates/sealed-secret.yaml
secure-sealed-secrets.sh
```

### 🏗️ CI/CD INTEGRATION:

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup kubeseal
        run: |
          wget https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.0/kubeseal-0.24.0-linux-amd64.tar.gz
          tar -xvzf kubeseal-0.24.0-linux-amd64.tar.gz
          sudo install -m 755 kubeseal /usr/local/bin/kubeseal
      
      - name: Create sealed secrets from vault
        env:
          DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
          OIDC_CLIENT_SECRET: ${{ secrets.OIDC_CLIENT_SECRET }}
          # ... other secrets from GitHub Secrets or vault
        run: |
          echo "$DB_PASSWORD" | kubeseal --raw --from-file=/dev/stdin --namespace=production --name=savannacart-release-secrets > db-password.sealed
          # ... encrypt other secrets
          
      - name: Deploy with Helm
        run: |
          helm upgrade --install savannacart-release ./savannacart --namespace production
```
