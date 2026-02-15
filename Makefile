.PHONY: build run analyze test clean deps hooks

VERSION ?= dev
BUILD_NUMBER ?= local
LDFLAGS = -X main.version=$(VERSION) -X main.buildNumber=$(BUILD_NUMBER)

build:
	go build -ldflags "$(LDFLAGS)" -o kfin

run: build
	./kfin status

analyze: build
	./kfin analyze

test:
	go test ./...

clean:
	rm -f kfin

deps:
	go mod download
	go mod tidy

hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit
	@echo "Git hooks installed from .githooks"
