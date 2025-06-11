# Script to generate sealed secrets for SavannaCart
# Run this script after installing sealed-secrets controller

param(
    [Parameter(Mandatory=$false)]
    [string]$Namespace = "savannacart"
)

Write-Host "🔐 Generating Sealed Secrets for SavannaCart" -ForegroundColor Green

# Check if kubeseal is available
if (-not (Get-Command "kubeseal" -ErrorAction SilentlyContinue)) {
    Write-Host "kubeseal command not found!" -ForegroundColor Red
    Write-Host "Please install kubeseal CLI from: https://github.com/bitnami-labs/sealed-secrets/releases" -ForegroundColor Yellow
    exit 1
}

# Check if sealed-secrets controller is running
Write-Host "🔍 Checking Sealed Secrets controller..." -ForegroundColor Yellow
try {
    kubectl get pods -n kube-system -l name=sealed-secrets-controller --no-headers 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Sealed Secrets controller not found!" -ForegroundColor Red
        Write-Host "Install it with: kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.0/controller.yaml" -ForegroundColor Yellow
        exit 1
    }
    Write-Host "Sealed Secrets controller is running" -ForegroundColor Green
} catch {
    Write-Host "rror checking Sealed Secrets controller" -ForegroundColor Red
    exit 1
}

# Function to create a sealed secret
function New-SealedSecret {
    param(
        [string]$Key,
        [string]$Value,
        [string]$SecretName,
        [string]$Namespace
    )
    
    Write-Host "  Encrypting $Key..." -ForegroundColor Cyan
    $encryptedValue = echo $Value | kubeseal --raw --from-file=/dev/stdin --namespace=$Namespace --name=$SecretName
    return $encryptedValue
}

# Prompt for secret values
Write-Host ">Please provide the secret values:" -ForegroundColor Yellow
Write-Host "   (Press Enter to use default values for development)" -ForegroundColor Gray

$secrets = @{}

# Database secrets
$secrets["db-user"] = Read-Host -Prompt "Database User [savannacart]"
if ([string]::IsNullOrEmpty($secrets["db-user"])) { $secrets["db-user"] = "savannacart" }

$secrets["db-password"] = Read-Host -Prompt "Database Password [savannacart-dev-password]" -AsSecureString
if ($secrets["db-password"].Length -eq 0) { 
    $secrets["db-password"] = ConvertTo-SecureString "savannacart-dev-password" -AsPlainText -Force 
}
$secrets["db-password"] = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secrets["db-password"]))

# JWT Secret
$secrets["jwt-secret"] = Read-Host -Prompt "JWT Secret [your-super-secret-jwt-key-change-in-production]"
if ([string]::IsNullOrEmpty($secrets["jwt-secret"])) { $secrets["jwt-secret"] = "your-super-secret-jwt-key-change-in-production" }

# OIDC secrets
$secrets["oidc-client-id"] = Read-Host -Prompt "OIDC Client ID [your-oidc-client-id]"
if ([string]::IsNullOrEmpty($secrets["oidc-client-id"])) { $secrets["oidc-client-id"] = "your-oidc-client-id" }

$secrets["oidc-client-secret"] = Read-Host -Prompt "OIDC Client Secret [your-oidc-client-secret]" -AsSecureString
if ($secrets["oidc-client-secret"].Length -eq 0) { 
    $secrets["oidc-client-secret"] = ConvertTo-SecureString "your-oidc-client-secret" -AsPlainText -Force 
}
$secrets["oidc-client-secret"] = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secrets["oidc-client-secret"]))

# SMTP secrets
$secrets["smtp-host"] = Read-Host -Prompt "SMTP Host [smtp.gmail.com]"
if ([string]::IsNullOrEmpty($secrets["smtp-host"])) { $secrets["smtp-host"] = "smtp.gmail.com" }

$secrets["smtp-port"] = Read-Host -Prompt "SMTP Port [587]"
if ([string]::IsNullOrEmpty($secrets["smtp-port"])) { $secrets["smtp-port"] = "587" }

$secrets["smtp-username"] = Read-Host -Prompt "SMTP Username [your-email@gmail.com]"
if ([string]::IsNullOrEmpty($secrets["smtp-username"])) { $secrets["smtp-username"] = "your-email@gmail.com" }

$secrets["smtp-password"] = Read-Host -Prompt "SMTP Password [your-app-password]" -AsSecureString
if ($secrets["smtp-password"].Length -eq 0) { 
    $secrets["smtp-password"] = ConvertTo-SecureString "your-app-password" -AsPlainText -Force 
}
$secrets["smtp-password"] = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secrets["smtp-password"]))

# SMS secrets
$secrets["sms-api-key"] = Read-Host -Prompt "SMS API Key [your-sms-api-key]"
if ([string]::IsNullOrEmpty($secrets["sms-api-key"])) { $secrets["sms-api-key"] = "your-sms-api-key" }

$secrets["sms-api-secret"] = Read-Host -Prompt "SMS API Secret [your-sms-api-secret]" -AsSecureString
if ($secrets["sms-api-secret"].Length -eq 0) { 
    $secrets["sms-api-secret"] = ConvertTo-SecureString "your-sms-api-secret" -AsPlainText -Force 
}
$secrets["sms-api-secret"] = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secrets["sms-api-secret"]))

# Generate sealed secrets
Write-Host "`nGenerating sealed secrets..." -ForegroundColor Yellow

$secretName = "savannacart-secrets"
$sealedSecrets = @{}

foreach ($key in $secrets.Keys) {
    Write-Host "  Encrypting $key..." -ForegroundColor Cyan
    try {
        $encryptedValue = echo $secrets[$key] | kubeseal --raw --from-file=/dev/stdin --namespace=$Namespace --name=$secretName
        $sealedSecrets[$key] = $encryptedValue.Trim()
    } catch {
        Write-Host "Failed to encrypt $key" -ForegroundColor Red
        exit 1
    }
}

# Generate the sealed secret YAML
$sealedSecretYaml = @"
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: $secretName
  namespace: $Namespace
spec:
  encryptedData:
    db-user: $($sealedSecrets['db-user'])
    db-password: $($sealedSecrets['db-password'])
    jwt-secret: $($sealedSecrets['jwt-secret'])
    oidc-client-id: $($sealedSecrets['oidc-client-id'])
    oidc-client-secret: $($sealedSecrets['oidc-client-secret'])
    smtp-host: $($sealedSecrets['smtp-host'])
    smtp-port: $($sealedSecrets['smtp-port'])
    smtp-username: $($sealedSecrets['smtp-username'])
    smtp-password: $($sealedSecrets['smtp-password'])
    sms-api-key: $($sealedSecrets['sms-api-key'])
    sms-api-secret: $($sealedSecrets['sms-api-secret'])
  template:
    metadata:
      name: $secretName
      namespace: $Namespace
    type: Opaque
"@

# Save to file
$outputFile = "savannacart-sealed-secrets.yaml"
$sealedSecretYaml | Out-File -FilePath $outputFile -Encoding UTF8

Write-Host "`Sealed secrets generated successfully!" -ForegroundColor Green
Write-Host "Saved to: $outputFile" -ForegroundColor Cyan
Write-Host "`To apply the sealed secrets:" -ForegroundColor Yellow
Write-Host " kubectl apply -f $outputFile" -ForegroundColor Cyan
Write-Host "` Replace the encryptedData values in your Helm chart templates with these values" -ForegroundColor Yellow
