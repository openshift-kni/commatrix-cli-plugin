GO_SRC := cmd/kubectl-commatrix.go
.DEFAULT_GOAL := run
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec
CURPATH=$(PWD)
BIN_DIR=$(CURPATH)/bin
BASH_SCRIPTS=$(shell find . -name "*.sh" -not -path "./.git/*")
EXECUTABLE_DIR=_output

GOLANGCI_LINT = $(BIN_DIR)/golangci-lint
# golangci-lint version should be updated periodically
# we keep it fixed to avoid it from unexpectedly failing on the project
# in case of a version bump
GOLANGCI_LINT_VER = v1.62.2

# Default locations for `make install`
INSTALL_DIR = /usr/local/bin

.PHONY: build
build:
	mkdir -p $(EXECUTABLE_DIR)
	go build -o $(EXECUTABLE_DIR)/kubectl-commatrix $(GO_SRC)

# Install the plugin and completion script in /usr/local/bin
.PHONY: install
install:
	install $(EXECUTABLE_DIR)/kubectl-commatrix $(INSTALL_DIR)

deps-update:
	go mod tidy

check-deps: deps-update
	@set +e; git diff --quiet HEAD go.sum go.mod; \
	if [ $$? -eq 1 ]; \
	then echo -e "\ngo modules are out of date. Please commit after running 'make deps-update' command\n"; \
	exit 1; fi

$(GOLANGCI_LINT): ; $(info installing golangci-lint...)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VER))

.PHONY: lint
lint: | $(GOLANGCI_LINT) ; $(info  running golangci-lint...) @ ## Run golangci-lint
	GOFLAGS="" $(GOLANGCI_LINT) run --timeout=10m


.PHONY: test
test:
	go test ./...

.PHONY: clean
clean:
	rm -f $(EXECUTABLE_DIR)/* $(BIN_DIR)/*

# go-install-tool will 'go install' any package $2 and install it to $1.
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOBIN=$(BIN_DIR) GOFLAGS="" go install $(2) ;\
}
endef