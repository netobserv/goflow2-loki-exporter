GOLANGCI_LINT_VERSION ?= v1.42.1
CONTAINER_COMMAND ?= docker
ARTIFACT_VERSION ?= $(shell git describe --long HEAD)

prereqs:
	@echo "### Test if prerequisites are met, and installing missing dependencies"
	test -f $(go env GOPATH)/bin/golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}

lint: prereqs
	@echo "### Linting code"
	golangci-lint run ./...

image:
	@echo "### Building container with ${CONTAINER_COMMAND}"
	${CONTAINER_COMMAND} build --build-arg VERSION=${ARTIFACT_VERSION} -t quay.io/jotak/goflow2:loki-latest .

.PHONY: prereqs lint build-container