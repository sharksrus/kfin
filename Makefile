.PHONY: build run test clean

build:
	go build -o pod-cost-analyzer

run: build
	./pod-cost-analyzer status

analyze: build
	./pod-cost-analyzer analyze

test:
	go test ./...

clean:
	rm -f pod-cost-analyzer

deps:
	go mod download
	go mod tidy
