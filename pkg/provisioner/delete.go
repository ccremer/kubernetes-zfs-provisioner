package provisioner

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	zfs "github.com/mistifyio/go-zfs"
	"k8s.io/client-go/pkg/api/v1"
)

// Delete removes a given volume from the server
func (p ZFSProvisioner) Delete(volume *v1.PersistentVolume) error {
	err := p.deleteVolume(volume)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"volume": volume.Spec.NFS.Path,
	}).Info("Deleted volume")
	return nil
}

// deleteVolume deletes a ZFS dataset from the server
func (p ZFSProvisioner) deleteVolume(volume *v1.PersistentVolume) error {
	dataset, err := zfs.GetDataset(p.zpool + "/" + p.parentDataset + "/" + volume.Name)
	if err != nil {
		return fmt.Errorf("Retrieving ZFS dataset for deletion failed with: %v", err.Error())
	}

	err = dataset.Destroy(zfs.DestroyRecursive)
	if err != nil {
		return fmt.Errorf("Deleting ZFS dataset failed with: %v", err.Error())
	}

	return nil
}
