#!/bin/bash

set -eo pipefail

zfs_bin=${ZFS_BIN:-zfs}
zfs_mod="${ZFS_MOD:-g+w}"

# Do not try to manually modify these Env vars, they will be updated by the provisioner just before invoking the script.
zfs_host="${ZFS_HOST}"
zfs_dataset="${ZFS_DATASET}"
zfs_update_permissions="${ZFS_UPDATE_PERMISSIONS:-yes}"

ssh "${zfs_host}" "${zfs_bin} ${*}"

if [[ ${zfs_update_permissions} == "yes" ]]; then
  mountpoint=$(ssh "${zfs_host}" "${zfs_bin} get -H -o value mountpoint ${zfs_dataset}")
  ssh "${zfs_host}" "chmod ${zfs_mod} ${mountpoint}" > /dev/null
fi
