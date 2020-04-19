#!/bin/bash

set -eo pipefail

zfs_bin=${ZFS_BIN:-zfs}

# Do not try to manually modify these Env vars, they will be updated by the provisioner just before invoking the script.
zfs_host="${ZFS_HOST}"

ssh "${zfs_host}" "${zfs_bin} ${*}"
