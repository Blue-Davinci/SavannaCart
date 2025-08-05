# SavannaCart Kubernetes Deployment Script
# This script sets up the SavannaCart application on a Kubernetes cluster using Helm

param(
    [Parameter(Mandatory=$false)]
    [string]$Environment = "development",
    
    [Parameter(Mandatory=$false)]
    [string]$Namespace = "savannacart",
    
    [Parameter(Mandatory=$false)]
    [switch]$UseSealedSecrets = $false,
    
    [Parameter(Mandatory=$false)]
    [switch]$InstallDependencies = $false
)

Write-Host "Starting SavannaCart Kubernetes Deployment" -ForegroundColor Green
Write-Host "Environment: $Environment" -ForegroundColor Cyan
Write-Host "Namespace: $Namespace" -ForegroundColor Cyan
Write-Host "Use Sealed Secrets: $UseSealedSecrets" -ForegroundColor Cyan

# Function to check if a command exists
function Test-Command {
    param($Command)
    try {
        Get-Command $Command -ErrorAction Stop
        return $true
    } catch {
        return $false
    }
}

# Check prerequisites
Write-Host "🔍 Checking prerequisites..." -ForegroundColor Yellow

$prerequisites = @("kubectl", "helm", "docker")
$missing = @()

foreach ($cmd in $prerequisites) {
    if (-not (Test-Command $cmd)) {
        $missing += $cmd
    } else {
        Write-Host "$cmd found" -ForegroundColor Green
    }
}

if ($missing.Count -gt 0) {
    Write-Host "Missing prerequisites: $($missing -join ', ')" -ForegroundColor Red
    Write-Host "Please install the missing tools and try again." -ForegroundColor Red
    exit 1
}

# Check if minikube is running
Write-Host "🔍 Checking Kubernetes cluster..." -ForegroundColor Yellow
try {
    $kubeContext = kubectl config current-context 2>$null
    Write-Host "Connected to cluster: $kubeContext" -ForegroundColor Green
} catch {
    Write-Host "No Kubernetes cluster available. Please start minikube or connect to a cluster." -ForegroundColor Red
    Write-Host "To start minikube: minikube start" -ForegroundColor Yellow
    exit 1
}

# Install dependencies if requested
if ($InstallDependencies) {
    Write-Host "Installing Kubernetes dependencies..." -ForegroundColor Yellow
    
    # Install NGINX Ingress Controller
    Write-Host "  Installing NGINX Ingress Controller..." -ForegroundColor Cyan
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml
    
    # Install Sealed Secrets Controller if requested
    if ($UseSealedSecrets) {
        Write-Host "  Installing Sealed Secrets Controller..." -ForegroundColor Cyan
        kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.0/controller.yaml
        
        # Check if kubeseal is installed
        if (-not (Test-Command "kubeseal")) {
            Write-Host "kubeseal CLI not found. Please install it manually:" -ForegroundColor Yellow
            Write-Host "  Download from: https://github.com/bitnami-labs/sealed-secrets/releases" -ForegroundColor Yellow
        }
    }
    
    Write-Host "Waiting for controllers to be ready..." -ForegroundColor Cyan
    Start-Sleep -Seconds 30
}

# Create namespace
Write-Host "Creating namespace: $Namespace" -ForegroundColor Yellow
kubectl create namespace $Namespace --dry-run=client -o yaml | kubectl apply -f -

# Build Docker image if not exists
Write-Host "Building Docker image..." -ForegroundColor Yellow
docker build -t savannacart/api:latest .

# Load image into minikube (if using minikube)
if ($kubeContext -like "*minikube*") {
    Write-Host "Loading image into minikube..." -ForegroundColor Yellow
    minikube image load savannacart/api:latest
}

# Deploy using Helm
Write-Host "Deploying SavannaCart with Helm..." -ForegroundColor Yellow

$helmArgs = @(
    "upgrade", "--install", "savannacart", "./savannacart",
    "--namespace", $Namespace,
    "--set", "image.repository=savannacart/api",
    "--set", "image.tag=latest",
    "--set", "app.environment=$Environment"
)

if (-not $UseSealedSecrets) {
    $helmArgs += "--set", "sealedSecrets.enabled=false"
}

# For development environment, use different values
if ($Environment -eq "development") {
    $helmArgs += @(
        "--set", "replicaCount=1",
        "--set", "autoscaling.enabled=false",
        "--set", "postgresql.primary.persistence.enabled=false",
        "--set", "ingress.hosts[0].host=api.savannacart.local"
    )
}

Write-Host "Running: helm $($helmArgs -join ' ')" -ForegroundColor Cyan
& helm @helmArgs

if ($LASTEXITCODE -eq 0) {
    Write-Host "Deployment successful!" -ForegroundColor Green
    
    # Get service information
    Write-Host "Getting service information..." -ForegroundColor Yellow
    kubectl get pods -n $Namespace
    kubectl get services -n $Namespace
    kubectl get ingress -n $Namespace
    
    # If using minikube, provide access instructions
    if ($kubeContext -like "*minikube*") {
        Write-Host "` == Access Instructions for Minikube:" -ForegroundColor Green
        Write-Host "1. Enable ingress addon: minikube addons enable ingress" -ForegroundColor Cyan
        Write-Host "2. Get minikube IP: minikube ip" -ForegroundColor Cyan
        Write-Host "3. Add to hosts file: <minikube-ip> api.savannacart.local" -ForegroundColor Cyan
        Write-Host "4. Access the API: http://api.savannacart.local/v1/api/healthcheck" -ForegroundColor Cyan
        
        Write-Host "`nOr use port-forward:" -ForegroundColor Green
        Write-Host "kubectl port-forward -n $Namespace service/savannacart 8080:80" -ForegroundColor Cyan
        Write-Host "Then access: http://localhost:8080/v1/api/healthcheck" -ForegroundColor Cyan
    }
    
} else {
    Write-Host "Deployment failed!" -ForegroundColor Red
    exit 1
}

Write-Host "`SavannaCart deployment completed!" -ForegroundColor Green
