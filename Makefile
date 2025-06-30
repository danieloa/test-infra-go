# Makefile for danieloa-infrastructure-go

.PHONY: all test lint deploy preview destroy clean build install

all: test lint build

# Build the deploy CLI tool
build:
	go build -o bin/deploy ./cmd/deploy

# Install the deploy CLI tool to $GOPATH/bin
install:
	go install ./cmd/deploy

# Run Go and Pulumi integration tests
test:
	go test ./...

# Lint Go code
lint:
	go fmt ./...
	go vet ./...

# Initialize S3 backend using CLI tool
init: build
	./bin/deploy init

# Deploy VPC stage using CLI tool
deploy-vpc: build
	./bin/deploy vpc --action up

# Deploy Core stage using CLI tool
deploy-core: build
	./bin/deploy core --action up

# Deploy Apps stage using CLI tool
deploy-apps: build
	./bin/deploy apps --action up

# Deploy all stages using CLI tool
deploy: build
	./bin/deploy vpc --action up
	./bin/deploy core --action up
	./bin/deploy apps --action up

# Pulumi preview using CLI tool
preview-vpc: build
	./bin/deploy vpc --action preview

preview-core: build
	./bin/deploy core --action preview

preview-apps: build
	./bin/deploy apps --action preview

# Destroy infrastructure using CLI tool
destroy-vpc: build
	./bin/deploy vpc --action down

destroy-core: build
	./bin/deploy core --action down

destroy-apps: build
	./bin/deploy apps --action down

destroy: build
	./bin/deploy apps --action down
	./bin/deploy core --action down
	./bin/deploy vpc --action down

# Clean build/test artifacts
clean:
	rm -rf bin/ *.test *.out
