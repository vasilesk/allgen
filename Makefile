tidy:
	go mod tidy

lint:
	golangci-lint run -c .golangci.yml ./...

lint-fix:
	golangci-lint run --fix -c .golangci.yml ./...

generate:
	go generate ./...

check-generated: generate
	git diff --exit-code

test:
	go test -race -coverprofile=cover.out -covermode=atomic ./...

coverage: test
	go tool cover -html=cover.out -o cover.html

build:
	go build -v ./...

.PHONY: tidy lint lint-fix generate check-generated test coverage build
