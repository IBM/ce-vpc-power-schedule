.PHONY: build build-image test run lint clean

BIN=bin/ce-vpc-power-schedule

build:
	go build -o $(BIN) ./cmd/ce-vpc-power-schedule

build-image:
	podman build -t ce-vpc-power-schedule:latest .

test:
	go test ./...

run:
	go run ./cmd/ce-vpc-power-schedule

lint:
	golangci-lint run || echo "Install golangci-lint to enable linting"

clean:
	rm -rf bin
