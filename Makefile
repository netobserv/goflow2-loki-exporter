COVERPROFILE ?= coverage.out
GOLANGCI_LINT_VERSION ?= v1.42.1

ARTIFACT_VERSION ?= $(shell git describe --long HEAD)
CONTAINER_COMMAND ?= docker

prereqs:
	@echo "### Test if prerequisites are met, and installing missing dependencies"
	test -f $(go env GOPATH)/bin/golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}

lint: prereqs
	@echo "### Linting code"
	golangci-lint run ./...

image:
	@echo "### Building container with ${CONTAINER_COMMAND}"
	${CONTAINER_COMMAND} build --build-arg VERSION=${ARTIFACT_VERSION} -t quay.io/jotak/goflow2:loki-latest .

test:
	echo "### Testing"
	go test ./... -coverprofile ${COVERPROFILE}

verify: lint test

.PHONY: prereqs lint image lint test verify