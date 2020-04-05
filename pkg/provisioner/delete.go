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
		return fmt.Errorf("Error retrieving dataset for deletion: %v", err)
	}

	err = dataset.Destroy(zfs.DestroyRecursive)
	if err != nil {
		return fmt.Errorf("Error destroying dataset: %v", err)
	}

	p.logger.Info("Deleted PV", zap.String("dataset", datasetPath))
	return nil
}
