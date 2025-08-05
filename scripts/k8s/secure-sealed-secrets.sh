#!/bin/bash

# SECURE Sealed Secrets Generator for SavannaCart
# This script prompts for secrets without storing them in plaintext

set -euo pipefail

NAMESPACE=${1:-"savannacart"}
SECRET_NAME="savannacart-release-secrets"

echo "🔐 Secure Sealed Secrets Generator for SavannaCart"
echo "Namespace: $NAMESPACE"
echo "Secret Name: $SECRET_NAME"
echo ""

# Check if kubeseal is available
if ! command -v kubeseal &> /dev/null; then
    echo "❌ kubeseal command not found!"
    echo "Install it from: https://github.com/bitnami-labs/sealed-secrets/releases"
    exit 1
fi

# Check if sealed-secrets controller is running
echo "🔍 Checking Sealed Secrets controller..."
if ! kubectl get pods -n kube-system -l name=sealed-secrets-controller --no-headers 2>/dev/null | grep -q Running; then
    echo "❌ Sealed Secrets controller not found or not running!"
    exit 1
fi
echo "✅ Sealed Secrets controller is running"

# Function to create a sealed secret
create_sealed_secret() {
    local key=$1
    local value=$2
    
    echo "🔒 Encrypting $key..." >&2
    printf "%s" "$value" | kubeseal --raw --from-file=/dev/stdin --namespace="$NAMESPACE" --name="$SECRET_NAME"
}

# Function to prompt for secret with hidden input
prompt_secret() {
    local prompt=$1
    local is_password=${2:-false}
    local value
    
    if [ "$is_password" = true ]; then
        echo -n "$prompt: " >&2
        read -s value
        echo >&2 # New line after hidden input
    else
        echo -n "$prompt: " >&2
        read value
    fi
    
    # Validate not empty
    if [ -z "$value" ]; then
        echo "❌ Value cannot be empty!" >&2
        exit 1
    fi
    
    echo "$value"
}

echo ""
echo "📝 Please provide the secret values:"
echo "   (All input will be hidden for passwords/tokens)"
echo ""

# Collect secrets securely
echo "=== Database Configuration ==="
DB_USER=$(prompt_secret "Database User")
DB_PASSWORD=$(prompt_secret "Database Password" true)

echo ""
echo "=== OAuth Configuration ==="
OIDC_CLIENT_ID=$(prompt_secret "OIDC Client ID")
OIDC_CLIENT_SECRET=$(prompt_secret "OIDC Client Secret" true)

echo ""
echo "=== SMTP Configuration ==="
SMTP_HOST=$(prompt_secret "SMTP Host")
SMTP_USERNAME=$(prompt_secret "SMTP Username")
SMTP_PASSWORD=$(prompt_secret "SMTP Password" true)
SMTP_SENDER=$(prompt_secret "SMTP Sender Email")

echo ""
echo "=== SMS Configuration ==="
SMS_ACCOUNT_SID=$(prompt_secret "SMS Account SID")
SMS_AUTH_TOKEN=$(prompt_secret "SMS Auth Token" true)
SMS_FROM_NUMBER=$(prompt_secret "SMS From Number")

echo ""
echo "🔄 Generating encrypted secrets..."

# Generate sealed secrets
SEALED_DB_USER=$(create_sealed_secret "db-user" "$DB_USER")
SEALED_DB_PASSWORD=$(create_sealed_secret "db-password" "$DB_PASSWORD")
SEALED_OIDC_CLIENT_ID=$(create_sealed_secret "oidc-client-id" "$OIDC_CLIENT_ID")
SEALED_OIDC_CLIENT_SECRET=$(create_sealed_secret "oidc-client-secret" "$OIDC_CLIENT_SECRET")
SEALED_SMTP_HOST=$(create_sealed_secret "smtp-host" "$SMTP_HOST")
SEALED_SMTP_USERNAME=$(create_sealed_secret "smtp-username" "$SMTP_USERNAME")
SEALED_SMTP_PASSWORD=$(create_sealed_secret "smtp-password" "$SMTP_PASSWORD")
SEALED_SMTP_SENDER=$(create_sealed_secret "smtp-sender" "$SMTP_SENDER")
SEALED_SMS_ACCOUNT_SID=$(create_sealed_secret "sms-account-sid" "$SMS_ACCOUNT_SID")
SEALED_SMS_AUTH_TOKEN=$(create_sealed_secret "sms-auth-token" "$SMS_AUTH_TOKEN")
SEALED_SMS_FROM_NUMBER=$(create_sealed_secret "sms-from-number" "$SMS_FROM_NUMBER")

# Clear variables from memory
unset DB_USER DB_PASSWORD OIDC_CLIENT_ID OIDC_CLIENT_SECRET
unset SMTP_HOST SMTP_USERNAME SMTP_PASSWORD SMTP_SENDER
unset SMS_ACCOUNT_SID SMS_AUTH_TOKEN SMS_FROM_NUMBER

# Create sealed secret manifest
SEALED_SECRET_FILE="$NAMESPACE-sealed-secrets.yaml"

cat > "$SEALED_SECRET_FILE" << EOF
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: $SECRET_NAME
  namespace: $NAMESPACE
  labels:
    app.kubernetes.io/name: savannacart
    app.kubernetes.io/instance: savannacart-release
spec:
  encryptedData:
    db-user: $SEALED_DB_USER
    db-password: $SEALED_DB_PASSWORD
    oidc-client-id: $SEALED_OIDC_CLIENT_ID
    oidc-client-secret: $SEALED_OIDC_CLIENT_SECRET
    smtp-host: $SEALED_SMTP_HOST
    smtp-username: $SEALED_SMTP_USERNAME
    smtp-password: $SEALED_SMTP_PASSWORD
    smtp-sender: $SEALED_SMTP_SENDER
    sms-account-sid: $SEALED_SMS_ACCOUNT_SID
    sms-auth-token: $SEALED_SMS_AUTH_TOKEN
    sms-from-number: $SEALED_SMS_FROM_NUMBER
  template:
    metadata:
      name: $SECRET_NAME
      namespace: $NAMESPACE
      labels:
        app.kubernetes.io/name: savannacart
        app.kubernetes.io/instance: savannacart-release
    type: Opaque
EOF

echo ""
echo "✅ Sealed secrets generated successfully!"
echo "📄 Sealed secret manifest saved to: $SEALED_SECRET_FILE"
echo ""
echo "🚀 To apply the sealed secret:"
echo "   kubectl apply -f $SEALED_SECRET_FILE"
echo ""
echo "🔍 To verify the secret was created:"
echo "   kubectl get secret $SECRET_NAME -n $NAMESPACE"
echo ""
echo "🧹 To clean up this script (recommended):"
echo "   rm $0"
