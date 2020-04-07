package provisioner

import (
	"fmt"
	"k8s.io/klog"
	"strconv"

	"github.com/mistifyio/go-zfs"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/controller"
)

// Provision creates a PersistentVolume, sets quota and shares it via NFS.
func (p ZFSProvisioner) Provision(options controller.ProvisionOptions) (*v1.PersistentVolume, error) {
	parameters, err := NewStorageClassParameters(options.StorageClass.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to parse StorageClass parameters: %w", err)
	}

	datasetPath := fmt.Sprintf("%s/%s", parameters.ParentDataset, options.PVName)
	properties := make(map[string]string)

	properties["sharenfs"] = fmt.Sprintf("rw=@%s%s", parameters.ShareSubnet, parameters.ShareOptions)

	storageRequest := options.PVC.Spec.Resources.Requests[v1.ResourceStorage]
	storageRequestBytes := strconv.FormatInt(storageRequest.Value(), 10)
	properties["refquota"] = storageRequestBytes
	properties["refreservation"] = storageRequestBytes

	dataset, err := zfs.CreateFilesystem(datasetPath, properties)
	if err != nil {
		return nil, fmt.Errorf("creating ZFS dataset failed: %w", err)
	}
	klog.Infof("dataset \"%s\": created", dataset.Name)

	// See nfs provisioner in github.com/kubernetes-incubator/external-storage for why we annotate this way and if it's still allowed
	annotations := make(map[string]string)
	annotations[annotationDatasetPathKey] = dataset.Name

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        options.PVName,
			Labels:      map[string]string{},
			Annotations: annotations,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: *options.StorageClass.ReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceStorage: options.PVC.Spec.Resources.Requests[v1.ResourceStorage],
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				NFS: &v1.NFSVolumeSource{
					Server:   parameters.Hostname,
					Path:     dataset.Mountpoint,
					ReadOnly: false,
				},
			},
		},
	}
	return pv, nil
}
