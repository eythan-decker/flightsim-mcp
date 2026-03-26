
BINARY_NAME=flightsim-mcp
IMAGE_NAME=ghcr.io/eythan-decker/flightsim-mcp
VERSION?=dev

build:
	go build -o bin/$(BINARY_NAME) ./cmd/flightsim-mcp

test:
	go test -race -count=1 ./...

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

vet:
	go vet ./...

fmt:
	gofmt -w .
	goimports -w .

clean:
	rm -rf bin/ coverage.out coverage.html

docker-build:
	docker build -t $(IMAGE_NAME):$(VERSION) -f deploy/docker/Dockerfile .

docker-push: docker-build
	docker push $(IMAGE_NAME):$(VERSION)

run: build
	@./bin/$(BINARY_NAME)

all: fmt vet lint test build
