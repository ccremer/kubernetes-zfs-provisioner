package provisioner

import (
	"fmt"
	"k8s.io/klog"

	"github.com/mistifyio/go-zfs"
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

	klog.V(2).Infof("dataset \"%s\": destroyed", dataset.Name)
	return nil
}
