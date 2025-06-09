<p align="center">
  <a href="" rel="noopener">
 <img width=200px height=200px src="https://i.imgur.com/6wj0hh6.jpg" alt="Project logo"></a>
</p>

<h3 align="center">SavannaCart</h3>

<div align="center">

[![Status](https://img.shields.io/badge/status-active-success.svg)]()
[![GitHub Issues](https://img.shields.io/github/issues/kylelobo/The-Documentation-Compendium.svg)](https://github.com/kylelobo/The-Documentation-Compendium/issues)
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
- [Usage](#usage)
- [Built Using](#built_using)
- [TODO](../TODO.md)
- [Contributing](../CONTRIBUTING.md)
- [Authors](#authors)
- [Acknowledgments](#acknowledgement)

## 🧐 About <a name = "about"></a>

**SavannaCart** is a secure and scalable backend service built in Go for managing a product catalog, customer orders, and OpenID Connect-based authentication. Designed for Savannah Informatics, it supports hierarchical product categories of arbitrary depth, order processing, SMS/email notifications, and caching using Redis.

The goal is to simulate a real-world microservice that handles e-commerce-style workflows with best practices in authentication, containerization, and test-driven development.

## 🏁 Getting Started <a name = "getting_started"></a>

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Prerequisites

Ensure the following are installed:

```
Go 1.21+
PostgreSQL 15+
Redis 6+
Docker & Docker Compose
Git
```

### Installing

Clone the repository and run:

```
git clone https://github.com/Blue-Davinci/savannacart.git
cd savannacart
cp .env.example .env
```

Run migrations:

```
make migrate
```

Start Redis and PostgreSQL (if using docker-compose):

```
docker-compose up -d
```

Run the server:

```
go run cmd/server/main.go
```

## 🔧 Running the tests <a name = "tests"></a>

### Unit tests

```
go test ./... -v
```

Tests cover:
- Token generation
- Authentication flow
- Redis caching logic
- Order creation handlers

### Linting

```
golangci-lint run
```

## 🎈 Usage <a name="usage"></a>

- Authenticate users via OpenID Connect
- Submit or fetch products and their nested categories
- Place orders and get email/SMS confirmations
- Retrieve average product prices for any category recursively
- Secure all APIs with bearer token middleware

## 🚀 Deployment <a name = "deployment"></a>

To deploy with Docker:

```
docker build -t savannacart .
docker run -p 8080:8080 --env-file .env savannacart
```

Kubernetes manifests are located in the `/deploy/k8s` directory for deployment via Minikube or kind.

## ⛏️ Built Using <a name = "built_using"></a>

- [Go (Chi)](https://github.com/go-chi/chi) - Web framework
- [Redis](https://redis.io/) - Caching and token/session store
- [PostgreSQL](https://www.postgresql.org/) - Main database
- [Docker](https://www.docker.com/) - Containerization
- [OIDC (coreos/go-oidc)](https://github.com/coreos/go-oidc) - Authentication
- [Africa's Talking](https://africastalking.com/) - SMS gateway
- [gomail](https://pkg.go.dev/gopkg.in/gomail.v2) - Email notifications
- [GitHub Actions](https://github.com/features/actions) - CI/CD automation

## ✍️ Authors <a name = "authors"></a>

- [@BrianKaricha](https://github.com/Blue-Davinci) - Design & Implementation

See also the list of [contributors](https://github.com/Blue-Davinci/savannacart/contributors) who participated in this project.

## 🎉 Acknowledgements <a name = "acknowledgement"></a>

- Savannah Informatics for the assessment prompt
- CoreOS OIDC team for Go integration tools
- Africa’s Talking for their developer sandbox
- PostgreSQL community for robust recursive querying support