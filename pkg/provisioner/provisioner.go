package provisioner

import (
	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"
	"sync"
)

const (
	DatasetPathAnnotation = "zfs.pv.kubernetes.io/zfs-dataset-path"
	ZFSHostAnnotation     = "zfs.pv.kubernetes.io/zfs-host"
	ZFSHostEnvVar         = "ZFS_HOST"
)

var (
	globalLock = sync.Mutex{}
)

// ZFSProvisioner implements the Provisioner interface to create and export ZFS volumes
type ZFSProvisioner struct {
	zfs zfs.Interface
}

// NewZFSProvisioner returns a new ZFSProvisioner based on a given optional
// zap.Logger. If none given, zaps default production logger is used.
func NewZFSProvisioner() (*ZFSProvisioner, error) {
	return &ZFSProvisioner{zfs: zfs.NewInterface()}, nil
}
