.PHONY: build run test clean

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
