SHELL := /bin/bash
BINARY := digitarlopanel
BACKEND := backend
FRONTEND := frontend
VERSION ?= 0.1.0

.PHONY: all build frontend backend dev test vet clean run release

## build: build the frontend then the single self-contained binary
build: frontend backend

## release: build the frontend then portable, stripped linux binaries (amd64+arm64)
## into dist/ with checksums. CGO-free, so cross-compiling needs no C toolchain.
release: frontend
	cd $(BACKEND) && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o ../dist/dp-amd64 ./cmd/digitarlopanel
	cd $(BACKEND) && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o ../dist/dp-arm64 ./cmd/digitarlopanel
	cd dist && gzip -9 -c dp-amd64 > digitarlopanel-linux-amd64.gz && gzip -9 -c dp-arm64 > digitarlopanel-linux-arm64.gz && rm -f dp-amd64 dp-arm64
	cd dist && sha256sum digitarlopanel-linux-amd64.gz digitarlopanel-linux-arm64.gz > SHA256SUMS

## frontend: install deps (if needed) and build the Vue app into backend/web/dist
frontend:
	cd $(FRONTEND) && npm install && npm run build

## backend: compile the Go binary with the embedded frontend
backend:
	cd $(BACKEND) && go build -ldflags "-X main.version=$(VERSION)" -o ../$(BINARY) ./cmd/digitarlopanel

## dev: run backend (:8088) and frontend dev server (:5173) — needs two terminals
dev:
	@echo "Run 'make backend && ./$(BINARY)' in one terminal and"
	@echo "'cd $(FRONTEND) && npm run dev' in another (proxy to :8088)."

## run: build and run the binary on :8088
run: build
	./$(BINARY) -config ./config.dev.json -listen :8088

## test: run backend unit tests
test:
	cd $(BACKEND) && go test ./...

## vet: run go vet
vet:
	cd $(BACKEND) && go vet ./...

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf $(BACKEND)/web/dist/assets
