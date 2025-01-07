GO_SRC := cmd/kubectl-commatrix.go
EXECUTABLE := kubectl-commatrix
.DEFAULT_GOAL := run
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec
CURPATH=$(PWD)
BIN_DIR=$(CURPATH)/bin
BASH_SCRIPTS=$(shell find . -name "*.sh" -not -path "./.git/*")


.PHONY: all build deps-update check-deps fmt-code lint lint-go lint-shell lint-md lint-sh test clean

# Default target
all: lint test

# Build the executable
build:
	go build -o $(EXECUTABLE) $(GO_SRC)

# Update dependencies
deps-update:
	go mod tidy

# Check if go modules are up to date
check-deps: deps-update
	@set +e; git diff --quiet HEAD go.sum go.mod; \
	if [ $$? -eq 1 ]; \
	then echo -e "\ngo modules are out of date. Please commit after running 'make deps-update' command\n"; \
	exit 1; fi

# Run go fmt against code
fmt-code:
	go fmt ./...

# Lint the project
lint: lint-go lint-shell lint-md lint-sh

# Run GolangCI-Lint
lint-go:
	checkmake --config=.checkmake Makefile
	golangci-lint run --timeout 10m0s

# Lint shell scripts
lint-shell:
	shfmt -d scripts/*.sh
	shellcheck --format=gcc ${BASH_SCRIPTS}

# Lint Markdown files
lint-md:
	typos
	markdownlint '**/*.md'

# Run tests
test:
	go test ./...

# Clean target to remove generated files
clean:
	rm -f $(EXECUTABLE) $(BIN_DIR)/*
