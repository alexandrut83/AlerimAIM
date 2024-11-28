.PHONY: build clean run test

# Build settings
BINARY_NAME=alerimnode
GO_FILES=$(shell find . -name '*.go')

# Build the application
build: $(GO_FILES)
	go build -o bin/$(BINARY_NAME) ./cmd/alerimnode

# Clean build artifacts
clean:
	rm -rf bin/

# Run the node
run: build
	./bin/$(BINARY_NAME)

# Run tests
test:
	go test -v ./...

# Run node with specific ports
run-node:
	./bin/$(BINARY_NAME) -port=8545 -p2p=9000

# Run additional nodes for testing
run-node2:
	./bin/$(BINARY_NAME) -port=8546 -p2p=9001 -peers=localhost:9000

run-node3:
	./bin/$(BINARY_NAME) -port=8547 -p2p=9002 -peers=localhost:9000,localhost:9001

# Build and deploy web wallet
deploy-wallet:
	mkdir -p /var/www/alerim
	cp -r wallet/web/* /var/www/alerim/
	chmod -R 755 /var/www/alerim
