PROJECT_DIR ?= $(shell pwd)
WORK_DIR = $(PROJECT_DIR)/.work

ZPOOL_SIZE=1 # in GB
zpool_dir := .zpool
zpool_disk := $(zpool_dir)/zpool.img
zpool_name_file := $(zpool_dir)/zpool.nfo
zpool_name := $(shell bash -c "cat .zpool/zpool.nfo || echo test$$RANDOM")
zfs_dataset := $(zpool_name)/zfs-provisioner

binary ?= kubernetes-zfs-provisioner

IMAGE_REGISTRY ?= ghcr.io
IMAGE_REPOSITORY ?= $(IMAGE_REGISTRY)/ccremer/zfs-provisioner
IMAGE_TAG ?= latest
