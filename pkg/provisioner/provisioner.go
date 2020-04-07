package provisioner

const (
	annotationDatasetPathKey = "gentics.com/zfs-dataset-path"
)

// ZFSProvisioner implements the Provisioner interface to create and export ZFS volumes
type ZFSProvisioner struct {

}

// NewZFSProvisioner returns a new ZFSProvisioner based on a given optional
// zap.Logger. If none given, zaps default production logger is used.
func NewZFSProvisioner() (*ZFSProvisioner, error) {
	return  &ZFSProvisioner{}, nil
}
