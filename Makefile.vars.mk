
ZPOOL_SIZE=1 # in GB
zpool_dir := .zpool
zpool_disk := $(zpool_dir)/zpool.img
zpool_name_file := $(zpool_dir)/zpool.nfo
zpool_name := $(shell bash -c "cat .zpool/zpool.nfo || echo test$$RANDOM")
zfs_dataset := $(zpool_name)/zfs-provisioner

binary ?= kubernetes-zfs-provisioner
