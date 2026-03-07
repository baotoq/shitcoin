.PHONY: test lint ci docker-build tilt-up kind-create kind-delete

# Run all Go tests
test:
	go test ./...

# Run golangci-lint
lint:
	golangci-lint run

# Run full CI checks locally (test + lint + frontend)
ci: test lint
	cd web && npm run lint && npm run build

# Build both Docker images
docker-build:
	docker build -t shitcoin-backend .
	docker build -t shitcoin-frontend web/

# Create kind cluster (idempotent)
kind-create:
	kind create cluster --config deploy/k8s/kind-cluster.yaml || true

# Delete kind cluster
kind-delete:
	kind delete cluster --name shitcoin

# Create kind cluster and start Tilt
tilt-up: kind-create
	tilt up
