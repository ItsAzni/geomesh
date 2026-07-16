.PHONY: build test run lint clean download-geoip

BINARY := bin/geomesh
CONFIG ?= examples/geomesh.yaml

build:
	go build -o $(BINARY) ./cmd/geomesh

test:
	go test ./... -v -count=1

test-race:
	go test -race ./... -count=1

run:
	go run ./cmd/geomesh $(CONFIG)

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/ test-fixtures/*.mmdb

download-geoip:
	@if [ -z "$(MAXMIND_LICENSE_KEY)" ]; then \
		echo "Error: set MAXMIND_LICENSE_KEY=<your-key>"; exit 1; \
	fi
	bash scripts/download-geoip.sh $(MAXMIND_LICENSE_KEY)
