package provisioner

const (
	annotationCreatedByKey   = "kubernetes.io/createdby"
	annotationDatasetPathKey = "gentics.com/zfs-dataset-path"
	createdBy                = "gentics.com/zfs"

	// Name is the provisoner name referenced in storage classes
	Name = "gentics.com/zfs"
)

// ZFSProvisioner implements the Provisioner interface to create and export ZFS volumes
type ZFSProvisioner struct {

}

// NewZFSProvisioner returns a new ZFSProvisioner based on a given optional
// zap.Logger. If none given, zaps default production logger is used.
func NewZFSProvisioner() (*ZFSProvisioner, error) {
	return  &ZFSProvisioner{}, nil
}
