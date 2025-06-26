# 🚀 SavannaCart Deployment Guide

## Prerequisites

- Docker and Docker Compose installed
- GitHub account with repository access
- Server with Docker for production deployment

## 🐳 Local Development with Docker

1. **Copy environment file:**
   ```bash
   cp .env.example .env
   # Edit .env with your actual values
   ```

2. **Start services:**
   ```bash
   docker-compose up -d
   ```

3. **View logs:**
   ```bash
   docker-compose logs -f api
   ```

4. **Stop services:**
   ```bash
   docker-compose down
   ```

## 🔄 GitHub Actions CI/CD

The repository includes automated CI/CD with GitHub Actions:

### Workflow Steps:
1. **Test** - Runs all Go tests with PostgreSQL
2. **Build** - Creates Docker image and pushes to GitHub Container Registry
3. **Deploy** - Deploys to production (main branch only)

### Required GitHub Secrets:
Set these in your repository settings under Secrets and Variables > Actions:

```
SAVANNACART_OIDC_CLIENT_ID
SAVANNACART_OIDC_CLIENT_SECRET
SAVANNACART_SMTP_HOST
SAVANNACART_SMTP_USERNAME
SAVANNACART_SMTP_PASSWORD
SAVANNACART_SMTP_SENDER
SAVANNACART_SMS_ACCOUNT_SID
SAVANNACART_SMS_AUTH_TOKEN
SAVANNACART_SMS_FROM_NUMBER
```

## 🏗️ Manual Docker Build

```bash
# Build image
docker build -t savannacart .

# Run container
docker run -p 4000:4000 \
  -e SAVANNACART_DB_DSN="your-db-connection" \
  savannacart
```

## 🌐 Production Deployment

1. **Server Setup:**
   ```bash
   # Install Docker and Docker Compose
   curl -fsSL https://get.docker.com -o get-docker.sh
   sh get-docker.sh
   ```

2. **Deploy:**
   ```bash
   # Make deploy script executable
   chmod +x deploy.sh
   
   # Run deployment
   ./deploy.sh
   ```

3. **Environment Variables:**
   Create a `.env` file on your server with production values.

## 📊 Health Checks

The application includes health check endpoints:
- **API Health**: `GET /v1/api/healthcheck`
- **Docker Health**: Built-in container health checks

## 🔧 Configuration

Key environment variables:
- `SAVANNACART_DB_DSN` - PostgreSQL connection string
- `SAVANNACART_OIDC_CLIENT_ID/SECRET` - Google OAuth
- `SAVANNACART_SMTP_*` - Email configuration
- `SAVANNACART_SMS_*` - Twilio SMS configuration

## 📝 Logs

View application logs:
```bash
# Docker Compose
docker-compose logs -f api

# Individual container
docker logs -f container_name
```
