package provisioner

import (
	"context"
	"fmt"
	"strconv"

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
	dataset := &zfs.Dataset{Name: datasetPath, Hostname: zfsHost}

	p.log.Info("deleting volume", "pv", volume.Name, "dataset", datasetPath, "host", zfsHost)

	if !destroySnapshotsAllowed(volume) {
		snaps, err := p.zfs.ListSnapshots(dataset)
		if err != nil {
			return fmt.Errorf("listing snapshots before destroy: %w", err)
		}
		if len(snaps) > 0 {
			return fmt.Errorf("refusing to destroy %s: %d snapshot(s) exist (e.g. %s); set StorageClass parameter destroySnapshots=true to override",
				datasetPath, len(snaps), snaps[0])
		}
	}

	err := p.zfs.DestroyDataset(dataset, zfs.DestroyRecursively)
	if err != nil {
		return fmt.Errorf("error destroying dataset: %w", err)
	}

	p.log.Info("dataset destroyed", "dataset", datasetPath, "pv", volume.Name)
	return nil
}

func destroySnapshotsAllowed(volume *core.PersistentVolume) bool {
	v := volume.Annotations[DestroySnapshotsAnnotation]
	if v == "" {
		// Historic behaviour: destroy -r, including snapshots.
		return true
	}
	ok, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return ok
}
