#!/bin/bash

set -eo pipefail

zfs_mod="${ZFS_MOD:-g+w}"
chmod_bin=${ZFS_CHOWN_BIN:-sudo -H chmod}

zfs_host="${1}"
zfs_mountpoint="${2}"

ssh "${zfs_host}" "${chmod_bin} ${zfs_mod} ${zfs_mountpoint}"
