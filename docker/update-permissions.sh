#!/bin/bash

set -eo pipefail

zfs_mod="${ZFS_MOD:-g+w}"
chmod_bin=${ZFS_CHOWN_BIN:-sudo -H chmod}

zfs_mountpoint="${1}"

# Do not try to manually modify these Env vars, they will be updated by the provisioner just before invoking the script.
zfs_host="${ZFS_HOST}"

ssh "${zfs_host}" "${chmod_bin} ${zfs_mod} ${zfs_mountpoint}"
