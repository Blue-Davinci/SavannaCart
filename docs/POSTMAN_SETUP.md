# SavannaCart API - Postman Configuration Guide

## 🎯 Recommended Setup for Postman

### Method 1: Direct IP Access (Easiest)
```
Base URL: https://192.168.49.2
Headers:
  Host: api.savannacart.local
  Content-Type: application/json
```

**Example Requests:**
- Health Check: `GET https://192.168.49.2/v1/api/healthcheck`
- Categories: `GET https://192.168.49.2/v1/api/categories`
- Products: `GET https://192.168.49.2/v1/api/products`

### Method 2: Local DNS (Production-like)
1. Add to `/etc/hosts`: `192.168.49.2 api.savannacart.local`
2. Use Base URL: `https://api.savannacart.local`

### Method 3: Port Forwarding (Development)
```bash
kubectl port-forward svc/savannacart-release -n savannacart 8080:80
```
Then use: `http://localhost:8080`

## 🔧 Postman Environment Variables

Create a Postman Environment with:
```json
{
  "name": "SavannaCart Minikube",
  "values": [
    {
      "key": "baseUrl",
      "value": "https://192.168.49.2",
      "enabled": true
    },
    {
      "key": "host",
      "value": "api.savannacart.local",
      "enabled": true
    },
    {
      "key": "namespace",
      "value": "savannacart",
      "enabled": true
    }
  ]
}
```

## 📝 Postman Collection Setup

### Pre-request Script (Collection Level):
```javascript
// Set the Host header for all requests
pm.request.headers.add({
    key: 'Host',
    value: pm.environment.get('host')
});

// Handle self-signed certificates
pm.settings.set("requestSSLCertificate", false);
```

### Example Request - Health Check:
```
Method: GET
URL: {{baseUrl}}/v1/api/healthcheck
Headers:
  Host: {{host}}
  Content-Type: application/json
```

## 🚀 Testing the Setup

1. **Health Check**: Should return `{"status": "API is healthy"}`
2. **Categories**: May require authentication
3. **Products**: May require authentication

## 🔐 Authentication Testing

If your API requires authentication, you'll need to:
1. Get OAuth token first
2. Include in Authorization header: `Bearer <token>`

## 🛠️ Troubleshooting

### SSL Certificate Issues:
- In Postman settings, turn off SSL certificate verification
- Or use `-k` flag with curl

### Connection Refused:
```bash
# Check if Minikube is running
minikube status

# Check if pods are running
kubectl get pods -n savannacart

# Check if ingress is working
kubectl get ingress -n savannacart
```

### DNS Resolution:
```bash
# Test direct IP access
curl -k -H "Host: api.savannacart.local" https://192.168.49.2/v1/api/healthcheck

# Test with local DNS
curl -k https://api.savannacart.local/v1/api/healthcheck
```
