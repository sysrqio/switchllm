.PHONY: build test lint run docker clean

BINARY := switchllm
PKG := ./...

build:
	go build -o bin/$(BINARY) ./cmd/switchllm

test:
	go test $(PKG) -count=1 -race

lint:
	go vet $(PKG)

run: build
	./bin/$(BINARY) start --dry-run --show-savings --config switchllm.yaml

docker:
	docker build -t switchllm:local .

clean:
	rm -rf bin/
