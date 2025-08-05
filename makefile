.PHONY: help
help:
	@echo "SavannaCart Build System"
	@echo "========================"
	@echo "Development:"
	@echo "  setup                - Set up development environment"
	@echo "  run/api              - Run the API server"
	@echo "  run/api/origins      - Run the API server with CORS origins"
	@echo "  db/psql              - Connect to the database using psql"
	@echo "  build/api            - Build the cmd/api application"
	@echo ""
	@echo "Docker:"
	@echo "  docker/build         - Build Docker image"
	@echo "  docker/run           - Run with docker-compose"
	@echo "  docker/down          - Stop docker-compose services"
	@echo ""
	@echo "Kubernetes:"
	@echo "  k8s/deploy           - Deploy to Kubernetes with Helm"
	@echo "  k8s/secrets          - Generate sealed secrets securely"
	@echo "  k8s/status           - Check deployment status"
	@echo "  k8s/clean            - Clean up Kubernetes resources"
	@echo ""
	@echo "Security:"
	@echo "  security/scan        - Scan for hardcoded secrets"
	@echo "  security/validate    - Validate security configuration"

.PHONY: run/api
run/api:
	@echo "Running SavannaCart API server..."
	go run ./cmd/api

.PHONY: run/api/origins
run/api/origins:
	@echo "Running API server with CORS origins..."
	go run ./cmd/api -cors-trusted-origins="http://localhost:5173"

# db/psql: connect to the db using psql
.PHONY: db/psql
db/psql:
	@echo "Connecting to the database using psql..."
	psql ${SAVANNACART_DB_DSN}

## build/api: build the cmd/api application
.PHONY: build/api
build/api:
	@echo "Building SavannaCart cmd/api..."
	go build -ldflags '-s' -o ./bin/api.exe ./cmd/api

## Docker targets
.PHONY: docker/build
docker/build:
	@echo "Building Docker image..."
	docker build -t savannacart:latest .

.PHONY: docker/run
docker/run:
	@echo "Starting services with docker-compose..."
	docker-compose up -d

.PHONY: docker/down
docker/down:
	@echo "Stopping docker-compose services..."
	docker-compose down

## Kubernetes targets
.PHONY: k8s/deploy
k8s/deploy:
	@echo "Deploying to Kubernetes with Helm..."
	./scripts/k8s/deploy.sh

.PHONY: k8s/secrets
k8s/secrets:
	@echo "Generating sealed secrets securely..."
	./scripts/k8s/secure-sealed-secrets.sh

.PHONY: k8s/status
k8s/status:
	@echo "Checking deployment status..."
	kubectl get pods,services,ingress -l app.kubernetes.io/name=savannacart

.PHONY: k8s/clean
k8s/clean:
	@echo "Cleaning up Kubernetes resources..."
	helm uninstall savannacart --ignore-not-found
	kubectl delete sealedsecrets savannacart-secrets --ignore-not-found=true

## Security targets
.PHONY: security/scan
security/scan:
	@echo "Scanning for hardcoded secrets..."
	@grep -r --exclude-dir=.git --exclude="*.example" -E "(password|secret|key|token).*=.*(pa55word|changeme|admin|root|test)" . || echo "No obvious hardcoded secrets found"

.PHONY: security/validate
security/validate:
	@echo "Validating security configuration..."
	@if [ ! -f .env.example ]; then echo "❌ Missing .env.example template"; else echo "✅ Environment template exists"; fi
	@if grep -q "pa55word\|admin\|root" .env.example; then echo "❌ Insecure defaults in .env.example"; else echo "✅ Secure .env.example"; fi
	@if [ -f .env ]; then echo "⚠️  .env file exists (should be in .gitignore)"; else echo "✅ No .env file committed"; fi

## Development setup
.PHONY: setup
setup:
	@echo "Setting up development environment..."
	./scripts/setup-dev.sh

## For linux builds: GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o bin/linux_amd64_api ./cmd/api