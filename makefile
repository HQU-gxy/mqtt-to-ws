.PHONY: all build clean run check cover lint docker help
BIN_FILE=bin
all: check build
build:
	@swag init
	@go build -o "${BIN_FILE}"
clean:
	@go clean
check:
	@go fmt ./
	@go vet ./
run:
	./"${BIN_FILE}"
lint:
	golangci-lint run --enable-all