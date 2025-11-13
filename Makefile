
# Check diff for generated files
.PHONY: check-diff
check-diff:
	/bin/bash -c 'if [[ -n "$$(git status --porcelain)" ]]; then \
		echo "ERROR: Generated files are out of date"; \
		echo ""; \
		echo "Please regenerate the files using the following commands:"; \
		echo "  - Backend API (Go clients): cd backend/api && make generate"; \
		echo "  - Frontend API (TypeScript clients): cd frontend && make generate-swagger-clients"; \
		echo "  - Kubernetes Platform: cd kubernetes_platform && make python-dev"; \
		echo "  - K8s CRDs: cd backend/src/crd/kubernetes && make generate manifests"; \
		echo ""; \
		echo "Changes found in the following files:"; \
		git status; \
		echo ""; \
		echo "Diff of changes:"; \
		git diff; \
		exit 1; \
	fi'

# Tools
BIN_DIR ?= $(CURDIR)/bin

.PHONY: ginkgo
ginkgo:
	mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install github.com/onsi/ginkgo/v2/ginkgo@latest
	@echo "Ginkgo installed to $(BIN_DIR)/ginkgo"
