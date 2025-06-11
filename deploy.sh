#!/bin/bash

# SavannaCart Deployment Script
# This script can be used to deploy the application to a server

set -e

echo "🚀 Starting SavannaCart deployment..."

# Configuration
IMAGE_NAME="ghcr.io/blue-davinci/savannacart:latest"
COMPOSE_FILE="docker-compose.prod.yml"

# Pull the latest image
echo "📦 Pulling latest Docker image..."
docker pull $IMAGE_NAME

# Stop existing services
echo "🛑 Stopping existing services..."
docker-compose -f $COMPOSE_FILE down

# Start new services
echo "▶️ Starting new services..."
docker-compose -f $COMPOSE_FILE up -d

# Wait for services to be healthy
echo "🔍 Waiting for services to be healthy..."
sleep 30

# Check health
echo "❤️ Checking service health..."
if curl -f http://localhost:4000/v1/api/healthcheck; then
    echo "✅ Deployment successful!"
else
    echo "❌ Deployment failed - rolling back..."
    docker-compose -f $COMPOSE_FILE down
    exit 1
fi

echo "🎉 SavannaCart deployed successfully!"
