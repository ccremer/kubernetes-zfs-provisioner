package provisioner

import (
	"context"
	"fmt"

	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"
	core "k8s.io/api/core/v1"
)

// Delete removes a given volume from the server
func (p *ZFSProvisioner) Delete(ctx context.Context, volume *core.PersistentVolume) error {
	for _, annotation := range []string{DatasetPathAnnotation, ZFSHostAnnotation} {
		value := volume.ObjectMeta.Annotations[annotation]
		if value == "" {
			return fmt.Errorf("annotation '%s' not found or empty, cannot determine which ZFS dataset to destroy", annotation)
		}
	}
	datasetPath := volume.ObjectMeta.Annotations[DatasetPathAnnotation]
	zfsHost := volume.ObjectMeta.Annotations[ZFSHostAnnotation]

	err := p.zfs.DestroyDataset(&zfs.Dataset{Name: datasetPath, Hostname: zfsHost}, zfs.DestroyRecursively)
	if err != nil {
		return fmt.Errorf("error destroying dataset: %w", err)
	}

	p.log.Info("dataset destroyed", "dataset", datasetPath)
	return nil
}
