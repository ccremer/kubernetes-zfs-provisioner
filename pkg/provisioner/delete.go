package provisioner

import (
	"fmt"

	"github.com/mistifyio/go-zfs"
	"go.uber.org/zap"
	core "k8s.io/api/core/v1"
)

// Delete removes a given volume from the server
func (p ZFSProvisioner) Delete(volume *core.PersistentVolume) error {
	datasetPath := volume.ObjectMeta.Annotations[annotationDatasetPathKey]
	dataset, err := zfs.GetDataset(datasetPath)
	if err != nil {
		return fmt.Errorf("error retrieving dataset for deletion: %w", err)
	}

	err = dataset.Destroy(zfs.DestroyRecursive)
	if err != nil {
		return fmt.Errorf("error destroying dataset: %w", err)
	}

	p.logger.Info("Deleted PV", zap.String("dataset", datasetPath))
	return nil
}
