SHELL := /usr/bin/env bash

# Disable built-in rules
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-builtin-variables
.SUFFIXES:
.SECONDARY:

include Makefile.vars.mk

.PHONY: help
help: ## Show this help
	@grep -E -h '\s##\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = "(: ).*?## "}; {gsub(/\\:/,":",$$1)}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Builds the binary
	go build ./...

.PHONY: install\:zfs
install\:zfs: ## Installs zfs-on-linux and nfs-kernel-server (requires sudo)
	sudo apt update
	sudo apt install -y zfsutils-linux nfs-kernel-server

$(zpool_dir):
	mkdir $(zpool_dir)

$(zpool_name_file): $(zpool_dir)
	# Create a disk image
	dd if=/dev/zero bs=1024M count=$(ZPOOL_SIZE) of=$$(pwd)/$(zpool_disk)
	echo $(zpool_name)
	sudo zpool create $(zpool_name) $$(pwd)/$(zpool_disk)
	echo "$(zpool_name)" > $(zpool_name_file)

/$(zfs_dataset): $(zpool_name_file)
	sudo zfs create $(zfs_dataset)
	sudo zfs allow -e create,destroy,snapshot,refreservation,refquota,share,sharenfs $(zfs_dataset)

.PHONY: prepare
prepare: /$(zfs_dataset) ## Prepares the zfs zpool for integration test

.PHONY: clean\:zfs
clean\:zfs: ## Cleans the zfs pool (requires sudo)
	sudo zpool destroy $(zpool_name)
	rm -rfv $(zpool_dir)

.PHONY: clean
clean: clean\:zfs ## Cleans everything
	rm -rf c.out $(binary) dist

.PHONY: test
test: ## Runs the unit tests
	go test -coverprofile c.out ./...

.PHONY: test\:integration
test\:integration: prepare ## Runs the integration tests with zfs (requires sudo)
	sudo sh -c "go env -w GOPATH=$$(go env GOPATH) && go test -v ./test/... -integration -parentDataset $(zfs_dataset)"
