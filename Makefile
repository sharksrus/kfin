.PHONY: build run analyze test clean deps hooks

build:
	go build -o kfin

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
