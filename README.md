# kubernetes-zfs-provisioner

zfs-provisioner is an out of cluster external provisioner for Kubernetes.
It creates ZFS datasets and shares them via NFS to make them mountable to pods.
Currently all ZFS attributes are inherited from the parent dataset, different storage
classes for e.g. cached/non-cached datasets or manually setting attributes via
annotations should follow in the future.

This provisioner is considered highly **experimental** and is still under development.

For more information about external storage in kubernetes, see [kubernetes-incubator/external-storage](https://github.com/kubernetes-incubator/external-storage).

## Usage

The provisioner can be configured via the following environment variables:

| Variable | Description | Default |
| :------: | :---------- | :-----: |
| `ZFS_METRICS_PORT` | Port on which to export Prometheus metrics. | `8080` |
| `ZFS_METRICS_ADDR` | Interface binding address on which to export Prometheus metrics. | `0.0.0.0` |
| `ZFS_KUBE_CONFIG_PATH` | Kubeconfig file path in which the credentials and API URL are defined. | `` |

Alternatively, provide a `zfs-provisioner.yaml` file in one of the following locations (first found):
* /etc/kubernetes
* /var/lib/kubernetes-zfs-provisioner (default file included in `deb` package)
* $PWD (current work dir)

See `packaging/zfs-provisioner.yaml` for the configuration options and defaults.

See Environment Variables for parameter description. Each entry can be overridden via Environment Variables, as they take
precedence over the values configured in the YAML file.

## Notes
### Reclaim policy

This provisioner currently supports the `Delete` and `Retain` reclaim policies.

### Storage space

The provisioner uses the `reflimit` and `refquota` ZFS attributes to limit
storage space for volumes. Each volume can not use more storage space than
the given resource request and also reserves exactly that much. This means
that over provisioning is not possible. Snapshots **do not** account for the
storage space limit.

See Oracles [ZFS Administration Guide](https://docs.oracle.com/cd/E23823_01/html/819-5461/gazvb.html) for more information.

## Development

### Requirements

* go 1.14
* goreleaser

The tests need to manage ZFS datasets, create a testing pool on a disk image:

```
# Create a 10GB disk image
dd if=/dev/zero bs=1024m count=10 of=disk.img
# Mount the image as a block device, MacOS way
hdiutil attach -imagekey diskimage-class=CRawDiskImage -nomount disk.img
# Create zpool with mount in current directory
sudo zpool create -m $PWD/test -f test /dev/disk2
```
For development under other operating systems, adapt mount command and block device.
