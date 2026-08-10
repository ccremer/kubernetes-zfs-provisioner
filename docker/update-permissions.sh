#!/bin/bash

set -eo pipefail

zfs_mod="${ZFS_MOD:-g+w}"
chmod_bin=${ZFS_CHOWN_BIN:-sudo -H chmod}

zfs_mountpoint="${1}"

# Do not try to manually modify these Env vars, they will be updated by the provisioner just before invoking the script.
zfs_host="${ZFS_HOST}"

# Quote the mountpoint so a path containing spaces or shell metacharacters is
# not re-split by the remote shell. chmod_bin/zfs_mod stay unquoted so
# "sudo -H chmod" and mode flags keep splitting into separate words.
ssh "${zfs_host}" "${chmod_bin} ${zfs_mod} $(printf '%q' "${zfs_mountpoint}")"
