package provisioner

import (
	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"
)

const (
	DatasetPathAnnotation      = "zfs.pv.kubernetes.io/zfs-dataset-path"
	ZFSHostAnnotation          = "zfs.pv.kubernetes.io/zfs-host"

	RefQuotaProperty           = "refquota"
	RefReservationProperty     = "refreservation"
	ManagedByProperty          = "io.kubernetes.pv.zfs:managed_by"
	ReclaimPolicyProperty      = "io.kubernetes.pv.zfs:reclaim_policy"
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
