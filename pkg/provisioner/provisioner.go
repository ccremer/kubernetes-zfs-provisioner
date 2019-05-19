package provisioner

import (
	"go.uber.org/zap"
)

const (
	annotationCreatedByKey   = "kubernetes.io/createdby"
	annotationDatasetPathKey = "gentics.com/zfs-dataset-path"
	createdBy                = "gentics.com/zfs"

	// Name is the provisoner name referenced in storage classes
	Name = "gentics.com/zfs"
)

// ZFSProvisioner implements the Provisioner interface to create and export ZFS volumes
type ZFSProvisioner struct {
	logger *zap.Logger
}

// NewZFSProvisioner returns a new ZFSProvisioner based on a given optional
// zap.Logger. If none given, zaps default production logger is used.
func NewZFSProvisioner(logger *zap.Logger) (*ZFSProvisioner, error) {
	var err error
	if logger == nil {
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	}
	provisioner := &ZFSProvisioner{
		logger,
	}

	return provisioner, nil
}
