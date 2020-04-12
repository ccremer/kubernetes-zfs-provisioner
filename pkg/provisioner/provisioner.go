package provisioner

import (
	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"
	"os"
	"sync"
)

const (
	DatasetPathAnnotation      = "zfs.pv.kubernetes.io/zfs-dataset-path"
	ZFSHostAnnotation          = "zfs.pv.kubernetes.io/zfs-host"
	ZFSHostEnvVar              = "ZFS_HOST"
	ZFSUpdatePermissionsEnvVar = "ZFS_UPDATE_PERMISSIONS"
	ZFSDatasetEnvVar           = "ZFS_DATASET"
	RefQuotaProperty           = "refquota"
	RefReservationProperty     = "refreservation"
	ManagedByProperty          = "io.kubernetes.pv.zfs:managed_by"
	ReclaimPolicyProperty      = "io.kubernetes.pv.zfs:reclaim_policy"
)

var (
	globalLock = sync.Mutex{}
)

// ZFSProvisioner implements the Provisioner interface to create and export ZFS volumes
type ZFSProvisioner struct {
	zfs          zfs.Interface
	InstanceName string
}

// NewZFSProvisioner returns a new ZFSProvisioner based on a given optional
// zap.Logger. If none given, zaps default production logger is used.
func NewZFSProvisioner(instanceName string) (*ZFSProvisioner, error) {
	return &ZFSProvisioner{
		zfs: zfs.NewInterface(), InstanceName: instanceName,
	}, nil
}

func setEnvironmentVars(hostName string, updatePermissions bool, zfsMountPath string) error {
	if err := os.Setenv(ZFSHostEnvVar, hostName); err != nil {
		return err
	}
	if updatePermissions {
		if err := os.Setenv(ZFSUpdatePermissionsEnvVar, "yes"); err != nil {
			return err
		}
	} else {
		if err := os.Setenv(ZFSUpdatePermissionsEnvVar, "no"); err != nil {
			return err
		}
	}
	if err := os.Setenv(ZFSDatasetEnvVar, zfsMountPath); err != nil {
		return err
	}
	return nil
}
