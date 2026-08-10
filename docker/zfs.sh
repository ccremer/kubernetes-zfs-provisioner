#!/bin/bash

set -eo pipefail

zfs_bin=${ZFS_BIN:-sudo -H zfs}

# Do not try to manually modify these Env vars, they will be updated by the provisioner just before invoking the script.
zfs_host="${ZFS_HOST}"

# Quote each argument individually so values that legitimately contain spaces
# or shell metacharacters survive parsing by the remote shell. Without this,
# "${*}" is flattened into one string and the remote shell re-splits it on
# whitespace: a multi-network "sharenfs=rw=@a ro=@b" would reach zfs(8) as two
# arguments and the create would fail. zfs_bin is intentionally left unquoted so
# "sudo -H zfs" keeps splitting into separate words.
remote_cmd="${zfs_bin}"
for arg in "$@"; do
	remote_cmd="${remote_cmd} $(printf '%q' "${arg}")"
done

ssh "${zfs_host}" "${remote_cmd}"
