.PHONY: all build clean run check cover lint docker help docs

BIN_FILENAME = bin

# https://stackoverflow.com/questions/4058840/makefile-that-distinguishes-between-windows-and-unix-like-systems
ifeq ($(OS),Windows_NT)
BIN_FILE=${BIN_FILENAME}.exe
else
BIN_FILE=${BIN_FILENAME}
endif

all: docs build
docs:
	@swag init
build:
	@go build -o "${BIN_FILE}"
clean:
	@go clean
check:
	@go fmt ./
	@go vet ./
run: build
	./"${BIN_FILE}"
lint:
	golangci-lint run --enable-all