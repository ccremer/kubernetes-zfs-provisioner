#!/bin/bash

set -eo pipefail

zfs_bin=${ZFS_BIN:-zfs}

ssh "${ZFS_HOST}" "${zfs_bin} ${*}"
