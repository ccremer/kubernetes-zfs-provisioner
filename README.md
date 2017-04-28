# kubernetes-zfs-provisioner

zfs-provisioner is an out of cluster external provisioner for Kubernetes. It creates ZFS datasets and shares them via NFS to make them mountable to pods. Currently all ZFS attributes are inherited from the parent dataset, different storage classes for e.g. cached/non-cached datasets or manually setting attributes via annotations should follow in the future. This provisioner is considered highly **experimental** and is still under development.

 For more information about external storage in kubernetes, see [kubernetes-incubator/external-storage](https://github.com/kubernetes-incubator/external-storage).

## Usage
The provisioner can be configured via the following environment variables:

| Variable | Description | Default |
| :------: | :---------- | :-----: |
| `ZFS_ZPOOL` | The zpool in which datasets will be created. | `storage` |
| `ZFS_ZPOOL_MOUNT_PREFIX` | The path under which the zpool is mounted. Usually not changed. | `/` |
| `ZFS_PARENT` | The parent dataset in which datasets will be created, needs to exist beforehand. No leading or trailing slashes. | `kubernetes/pv` |
| `ZFS_SHARE_SUBNET` | The subnet to which volumes will be exported. | `10.0.0.0/8` |
| `ZFS_SHARE_OPTIONS` | Additional nfs share options, comma-separated. | |
| `ZFS_HOSTNAME` | The hostname or ip which the pods should use to mount the volume. Determined via `hostname -f` if empty. | |
| `ZFS_KUBE_CONF` | Path to the kubernetes config file which will be used to connect to the cluster. |`kube.conf` |

## Development

The tests need to manage ZFS datasets, create a testing pool on a disk image:

```
# Create a 10GB disk image
dd if=/dev/zero bs=1024m count=10 of=disk.img
# Mount the image as a block device, MacOS way
hdiutil attach -imagekey diskimage-class=CRawDiskImage -nomount disk.img
# Create zpool with mount in current directory
sudo zpool create -m (pwd)/test -f test /dev/disk2
```
For development under other operating systems, adapt mount command and block device. 