<p align="center">
  <a href="" rel="noopener">
 <img width=200px height=200px src="https://i.ibb.co/Rpq9Tvwy/savanna-cart-high-resolution-logo-photoaidcom-cropped.png" alt="Project logo"></a>
</p>

<h3 align="center">SavannaCart</h3>

<div align="center">

[![Status](https://img.shields.io/badge/status-active-success.svg)]()
- 🏢 **Savannah Informatics** for the comprehensive assessment requirements
- 🔐 **CoreOS OIDC Team** for excellent Go OpenID Connect integration tools
- 📱 **Twilio** for reliable SMS API and developer-friendly documentation
- 🗄️ **PostgreSQL Community** for robust recursive querying and JSON support
- ☸️ **Kubernetes Community** for container orchestration excellence
- 🐳 **Docker** for revolutionizing application containerization
- 🔧 **Go Community** for building a fantastic ecosystem of tools and libraries

Special thanks to all open-source contributors who made this project possible! 🙏itHub Issues](https://img.shields.io/github/issues/kylelobo/The-Documentation-Compendium.svg)](https://github.com/kylelobo/The-Documentation-Compendium/issues)
[![GitHub Pull Requests](https://img.shields.io/github/issues-pr/kylelobo/The-Documentation-Compendium.svg)](https://github.com/kylelobo/The-Documentation-Compendium/pulls)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](/LICENSE)

</div>

---

<p align="center"> Currently in quick development. 
    <br> 
</p>

## 📝 Table of Contents

- [About](#about)
- [Getting Started](#getting_started)
- [Deployment](#deployment)
- [Kubernetes Deployment](#kubernetes)
- [Usage](#usage)
- [Testing](#tests)
- [Built Using](#built_using)
- [Authors](#authors)
- [Acknowledgments](#acknowledgement)

## 🧐 About <a name = "about"></a>

**SavannaCart** is a secure and scalable backend service built in Go for managing a product catalog, customer orders, and OpenID Connect-based authentication. Designed for Savannah Informatics, it supports hierarchical product categories of arbitrary depth, order processing, SMS/email notifications, and caching using Redis.

The application features:
- **OpenID Connect Authentication** with Google OAuth integration
- **Hierarchical Product Categories** with unlimited nesting depth
- **Order Management System** with email/SMS notifications
- **Production-Ready Deployment** with Docker and Kubernetes
- **Security-First Design** with proper secret management
- **Health Monitoring** with comprehensive health checks
- **Container Security** with non-root user execution

The goal is to demonstrate enterprise-grade microservice development with best practices in authentication, containerization, Kubernetes deployment, and production monitoring.

## 🏁 Getting Started <a name = "getting_started"></a>

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Prerequisites

Ensure the following are installed:

```
Go 1.23+
PostgreSQL 15+
Redis 6+
Docker & Docker Compose
Git
kubectl (for Kubernetes deployment)
Helm 3.x (for Kubernetes deployment)
Minikube (for local Kubernetes testing)
```

### Installing

Clone the repository and run:

```bash
git clone https://github.com/Blue-Davinci/savannacart.git
cd savannacart
cp .env.example .env
```

Configure your environment variables in `.env`:

```bash
# Database Configuration
SAVANNACART_DB_DSN=postgres://savannacart:pa55word@localhost/savannacart?sslmode=disable

# OAuth Configuration (replace with your Google OAuth credentials)
SAVANNACART_OIDC_CLIENT_ID=your-google-oauth-client-id
SAVANNACART_OIDC_CLIENT_SECRET=your-google-oauth-client-secret

# SMTP Configuration (for email notifications)
SAVANNACART_SMTP_HOST=smtp.gmail.com
SAVANNACART_SMTP_USERNAME=your-email@gmail.com
SAVANNACART_SMTP_PASSWORD=your-app-password
SAVANNACART_SMTP_SENDER=noreply@yourdomain.com

# SMS Configuration (Twilio)
SAVANNACART_SMS_ACCOUNT_SID=your-twilio-account-sid
SAVANNACART_SMS_AUTH_TOKEN=your-twilio-auth-token
SAVANNACART_SMS_FROM_NUMBER=+1234567890
```

Run database migrations:

```bash
cd internal/sql/schema
goose postgres "your-connection-string" up
```

Start dependencies with Docker Compose:

```bash
docker-compose up -d
```

Run the server:

```bash
go run cmd/api/main.go
```

The API will be available at `http://localhost:4000`

## 🔧 Running the tests <a name = "tests"></a>

### Unit Tests

Run the complete test suite:

```bash
go test ./... -v
```

Run tests with coverage:

```bash
go test ./... -v -cover
```

Tests cover:
- Authentication flow and token validation
- Order creation and management
- Category hierarchy operations
- Redis caching logic
- Email and SMS notification systems
- Security middleware and validation

### Linting

Ensure code quality with:

```bash
golangci-lint run
```

### Integration Tests

End-to-end API tests:

```bash
go test ./cmd/api -v -tags=integration
```

## 🎈 Usage <a name="usage"></a>

### API Endpoints

The SavannaCart API provides the following functionality:

#### 🔐 Authentication
- **OAuth Login**: `/v1/api/authentication` - Google OAuth integration
- **Token Validation**: Protected endpoints require Bearer token authentication

#### 📦 Products & Categories
- **List Categories**: `GET /v1/api/categories` - Retrieve hierarchical categories
- **Create Category**: `POST /v1/api/categories` - Add new product categories
- **List Products**: `GET /v1/api/products` - Browse product catalog
- **Create Product**: `POST /v1/api/products` - Add new products

#### 🛒 Orders
- **Create Order**: `POST /v1/api/orders` - Place new orders
- **Get Orders**: `GET /v1/api/orders` - Retrieve user orders
- **Order Status**: Email and SMS notifications for order updates

#### 📊 Monitoring
- **Health Check**: `GET /v1/api/healthcheck` - Service health status
- **Metrics**: `GET /debug/vars` - Application metrics and statistics

### Example API Usage

```bash
# Health check
curl http://localhost:4000/v1/api/healthcheck

# List categories (with authentication)
curl -H "Authorization: Bearer your-token" \
     http://localhost:4000/v1/api/categories

# Create a product
curl -X POST \
     -H "Authorization: Bearer your-token" \
     -H "Content-Type: application/json" \
     -d '{"name":"Product Name","price":99.99,"category_id":1}' \
     http://localhost:4000/v1/api/products
```

## 🚀 Deployment <a name = "deployment"></a>

### Docker Deployment

Build and run with Docker:

```bash
# Build the image
docker build -t savannacart/api:latest .

# Run with environment file
docker run -p 4000:4000 --env-file .env savannacart/api:latest
```

### Docker Compose

For development with all dependencies:

```bash
docker-compose up -d
```

This starts:
- PostgreSQL database
- Redis cache [**To be implemented**]
- SavannaCart API server

## ☸️ Kubernetes Deployment <a name = "kubernetes"></a>

### Prerequisites for Kubernetes

```bash
# Install required tools
minikube start
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
```

### Quick Deployment with Helm

```bash
# Deploy to Minikube
helm install savannacart-release ./savannacart -n savannacart --create-namespace

# Check deployment status
kubectl get pods -n savannacart

# Access the application
minikube service savannacart-release -n savannacart
```

### Production Deployment

For production environments:

```bash
# Enable sealed secrets for secure secret management
helm upgrade savannacart-release ./savannacart -n savannacart \
  --set sealedSecrets.enabled=true \
  --set image.tag=latest

# Scale the deployment
kubectl scale deployment savannacart-release --replicas=3 -n savannacart
```

### Deployment Features

- **Security**: Non-root container execution, read-only filesystem
- **Scaling**: Horizontal Pod Autoscaler (HPA) configured
- **Health Checks**: Liveness and readiness probes
- **Config Management**: ConfigMaps and Secrets for configuration
- **Secret Management**: Sealed Secrets for production security
- **Monitoring**: Built-in metrics and health endpoints

### Update Workflow

```bash
# 1. Build new image
docker build -t savannacart/api:v1.1 --no-cache .

# 2. Load into Minikube (for local development)
minikube image load savannacart/api:v1.1

# 3. Update deployment
helm upgrade savannacart-release ./savannacart -n savannacart --set image.tag=v1.1

# 4. Monitor rollout
kubectl rollout status deployment/savannacart-release -n savannacart
```

For detailed Kubernetes deployment instructions, see [K8S-DEPLOYMENT.md](K8S-DEPLOYMENT.md).

## ⛏️ Built Using <a name = "built_using"></a>

### Core Technologies
- [Go](https://golang.org/) - High-performance backend language
- [Chi Router](https://github.com/go-chi/chi) - Lightweight, idiomatic HTTP router
- [PostgreSQL](https://www.postgresql.org/) - Primary database with ACID compliance
- [Redis](https://redis.io/) - Caching and session management
- [SQLC](https://sqlc.dev/) - Type-safe SQL query generation

### Authentication & Security
- [OIDC (coreos/go-oidc)](https://github.com/coreos/go-oidc) - OpenID Connect authentication
- [OAuth2](https://pkg.go.dev/golang.org/x/oauth2) - Google OAuth integration
- [Zap](https://github.com/uber-go/zap) - Structured, high-performance logging

### Notifications
- [Twilio](https://www.twilio.com/) - SMS messaging service
- [Gomail](https://pkg.go.dev/gopkg.in/gomail.v2) - Email notification system

### DevOps & Deployment
- [Docker](https://www.docker.com/) - Containerization platform
- [Kubernetes](https://kubernetes.io/) - Container orchestration
- [Helm](https://helm.sh/) - Kubernetes package manager
- [Minikube](https://minikube.sigs.k8s.io/) - Local Kubernetes development
- [Sealed Secrets](https://sealed-secrets.netlify.app/) - Kubernetes secret encryption

### Development Tools
- [Air](https://github.com/cosmtrek/air) - Live reload for development
- [golangci-lint](https://golangci-lint.run/) - Fast Go linters runner
- [Goose](https://github.com/pressly/goose) - Database migration tool

### Infrastructure
- [GitHub Actions](https://github.com/features/actions) - CI/CD automation
- [Docker Compose](https://docs.docker.com/compose/) - Multi-container development

## ✍️ Authors <a name = "authors"></a>

- [@Blue](https://github.com/Blue-Davinci) - Design & Implementation

See also the list of [contributors](https://github.com/Blue-Davinci/savannacart/contributors) who participated in this project.

## 🎉 Acknowledgements <a name = "acknowledgement"></a>

- Savannah Informatics for the assessment prompt
- CoreOS OIDC team for Go integration tools
- Africa’s Talking for their developer sandbox
- PostgreSQL community for robust recursive querying support