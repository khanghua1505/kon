BINARY := kon

.PHONY: all build test lint fmt vet check clean install

all: build

build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(BINARY) ./cmd/kon

test:
	go test -cover -count=1 ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

check: vet lint test

clean:
	rm -f $(BINARY)

install:
	go install ./cmd/kon
