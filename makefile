.PHONY: all build clean run check cover lint docker help

# https://stackoverflow.com/questions/4058840/makefile-that-distinguishes-between-windows-and-unix-like-systems
ifeq ($(OS),Windows_NT)
BIN_FILE=bin.exe
else
BIN_FILE=bin
endif

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