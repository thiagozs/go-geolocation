SHELL := /bin/sh

APP_NAME ?= geolocation
DOCKER_IMAGE ?= $(APP_NAME):latest
RUN_CMD ?= runserver --http=5000

GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: help
help: ## Show available make targets
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z0-9_\-]+:.*##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^\.PHONY/ { next }' $(MAKEFILE_LIST)

.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy

.PHONY: fmt
fmt: ## Format go files
	gofmt -w $(GOFILES)

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: test
test: ## Run unit tests
	GOCACHE=$$(mktemp -d) go test ./...

.PHONY: run
run: ## Run server locally
	go run ./cmd/geolocation $(RUN_CMD)

.PHONY: build
build: ## Build binary into ./bin
	GOOS=$${GOOS:-$$(go env GOOS)} GOARCH=$${GOARCH:-$$(go env GOARCH)} go build -o bin/$(APP_NAME) .

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin

.PHONY: docker-build
docker-build: ## Build docker image
	docker build -t $(DOCKER_IMAGE) .

.PHONY: docker-run
docker-run: ## Run docker image and expose port 5000
	docker run --rm -p 5000:5000 $(DOCKER_IMAGE) $(RUN_CMD)

.PHONY: docker-push
docker-push: ## Push docker image to registry
	docker push $(DOCKER_IMAGE)

.PHONY: lint
lint: fmt vet ## Format and vet code

.PHONY: all
all: tidy fmt vet test build ## Run full pipeline
