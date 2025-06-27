# Makefile for danieloa-infrastructure-go

.PHONY: all test lint deploy preview destroy clean

all: test lint

# Run Go and Pulumi integration tests
test:
	go test ./...

# Lint Go code
lint:
	go fmt ./...
	go vet ./...

# Pulumi preview (dry-run)
preview:
	pulumi preview

# Pulumi deploy (apply changes)
deploy:
	pulumi up --yes

# Pulumi destroy (tear down infra)
destroy:
	pulumi destroy --yes

# Clean build/test artifacts
clean:
	rm -rf bin/ *.test *.out
